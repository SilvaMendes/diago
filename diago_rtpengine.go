package diago

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	"github.com/rs/zerolog/log"
)

// NewDiago construct b2b user agent that will act as server and client
func NewDiagoEngine(ua *sipgo.UserAgent, opts ...DiagoOption) *Diago {
	dg := &Diago{
		ua:  ua,
		log: log.Logger,
		serveHandler: func(d *DialogServerSession) {
			fmt.Println("Serve Handler not implemented")
		},
		transports: []Transport{},
		mediaConf:  MediaConfig{},
	}

	for _, o := range opts {
		o(dg)
	}

	if dg.client == nil {
		dg.client, _ = sipgo.NewClient(ua,
			sipgo.WithClientNAT(),
		)
	}

	if dg.server == nil {
		dg.server, _ = sipgo.NewServer(ua)
	}

	if len(dg.transports) == 0 {
		dg.transports = append(dg.transports, Transport{
			Transport:    "udp",
			BindHost:     "127.0.0.1",
			BindPort:     5060,
			ExternalHost: "127.0.0.1",
			ExternalPort: 5060,
		})
	}

	// Create our default contact hdr
	contactHDR := dg.getContactHDR("")

	server := dg.server
	server.OnInvite(func(req *sip.Request, tx sip.ServerTransaction) {
		// What if multiple server transports?
		id, err := sip.UASReadRequestDialogID(req)
		if err == nil {
			dg.handleReInviteRngine(req, tx, id)
			return
		}

		// Proceed as new call
		dialogUA := sipgo.DialogUA{
			Client:     dg.client,
			ContactHDR: contactHDR,
		}

		dialog, err := dialogUA.ReadInvite(req, tx)
		if err != nil {
			dg.log.Error().Err(err).Msg("Handling new INVITE failed")
			return
		}

		// TODO authentication
		dWrap := &DialogServerSession{
			DialogServerSession: dialog,
			DialogMedia:         DialogMedia{},
		}
		dg.initServerSession(dWrap)
		defer dWrap.Close()

		DialogsServerCache.DialogStore(dWrap.Context(), dWrap.ID, dWrap)
		defer func() {
			// TODO: have better context
			DialogsServerCache.DialogDelete(context.Background(), dWrap.ID)
		}()

		dg.serveHandler(dWrap)

		// Check is dialog closed
		dialogCtx := dialog.Context()
		// Always try hanguping call
		ctx, cancel := context.WithTimeout(dialogCtx, 10*time.Second)
		defer cancel()

		if err := dWrap.Hangup(ctx); err != nil {
			if errors.Is(ctx.Err(), context.Canceled) {
				// Already hangup
				return
			}

			dg.log.Error().Err(err).Msg("Hanguping call failed")
			return
		}
	})

	server.OnCancel(func(req *sip.Request, tx sip.ServerTransaction) {
		// INVITE transaction should be terminated by transaction layer and 200 response will be sent
		// In case of stateless proxy this we would need to forward
		tx.Respond(sip.NewResponseFromRequest(req, sip.StatusCallTransactionDoesNotExists, "Call/Transaction Does Not Exist", nil))
	})

	server.OnAck(func(req *sip.Request, tx sip.ServerTransaction) {
		d, err := MatchDialogServer(req)
		if err != nil {
			// Normally ACK will be received if some out of dialog request is received or we responded negatively
			// tx.Respond(sip.NewResponseFromRequest(req, sip.StatusBadRequest, err.Error(), nil))
			return
		}

		if err := d.ReadAck(req, tx); err != nil {
			dg.log.Error().Err(err).Msg("ACK finished with error")
			// Do not respond bad request as client will DOS on any non 2xx response
			return
		}
	})

	server.OnBye(func(req *sip.Request, tx sip.ServerTransaction) {
		sd, cd, err := MatchDialog(req)
		if err != nil {
			if errors.Is(err, sipgo.ErrDialogDoesNotExists) {
				tx.Respond(sip.NewResponseFromRequest(req, sip.StatusCallTransactionDoesNotExists, err.Error(), nil))
				return

			}
			tx.Respond(sip.NewResponseFromRequest(req, sip.StatusBadRequest, err.Error(), nil))
			return
		}

		// Respond to BYE
		// Terminate our media processing
		// As user may stuck in playing or reading media, this unblocks that goroutine
		if cd != nil {
			if err := cd.ReadBye(req, tx); err != nil {
				dg.log.Error().Err(err).Msg("failed to read bye for client dialog")
			}

			cd.DialogMedia.Close()
			return
		}

		if err := sd.ReadBye(req, tx); err != nil {
			dg.log.Error().Err(err).Msg("failed to read bye for server dialog")
		}
		sd.DialogMedia.Close()
	})

	server.OnInfo(func(req *sip.Request, tx sip.ServerTransaction) {
		// Handle DTMF out of band
		if req.ContentType().Value() != "application/dtmf-relay" {
			tx.Respond(sip.NewResponseFromRequest(req, sip.StatusNotAcceptable, "Not Acceptable", nil))
			return
		}

		sd, cd, err := MatchDialog(req)
		if err != nil {
			if errors.Is(err, sipgo.ErrDialogDoesNotExists) {
				tx.Respond(sip.NewResponseFromRequest(req, sip.StatusCallTransactionDoesNotExists, err.Error(), nil))
				return

			}
			tx.Respond(sip.NewResponseFromRequest(req, sip.StatusBadRequest, err.Error(), nil))
			return
		}

		if cd != nil {
			cd.readSIPInfoDTMF(req, tx)
			return
		}
		sd.readSIPInfoDTMF(req, tx)

		// 		INFO sips:sipgo@127.0.0.1:5443 SIP/2.0
		// Via: SIP/2.0/WSS df7jal23ls0d.invalid;branch=z9hG4bKhzJuRuWp4pLmTAbrIg7MUGofWdV1u577;rport
		// From: "IVR Webrtc"<sips:ivr.699c4b45-c800-4891-8133-fded5b26f942.579938@localhost:6060>;tag=foSxtEhHq9QOSeSdgJCC
		// To: <sip:playback@localhost>;tag=f814097f-467a-46ad-be0a-76c2a1225378
		// Contact: "IVR Webrtc"<sips:ivr.699c4b45-c800-4891-8133-fded5b26f942.579938@df7jal23ls0d.invalid;rtcweb-breaker=no;click2call=no;transport=wss>;+g.oma.sip-im;language="en,fr"
		// Call-ID: 047c3631-e85a-27d2-8f69-4de6e0391253
		// CSeq: 29586 INFO
		// Content-Type: application/dtmf-relay
		// Content-Length: 22
		// Max-Forwards: 70
		// User-Agent: IM-client/OMA1.0 sipML5-v1.2016.03.04

		// Signal=8
		// Duration=120

	})

	// TODO deal with OPTIONS more correctly
	// For now leave it for keep alive
	dg.server.OnOptions(func(req *sip.Request, tx sip.ServerTransaction) {
		res := sip.NewResponseFromRequest(req, sip.StatusOK, "OK", nil)
		if err := tx.Respond(res); err != nil {
			log.Error().Err(err).Msg("OPTIONS 200 failed to respond")
		}
	})

	return dg
}

func (dg *Diago) handleReInviteRngine(req *sip.Request, tx sip.ServerTransaction, id string) {
	ctx := context.TODO()
	// No Error means we have ID
	s, err := DialogsServerCache.DialogLoad(ctx, id)
	if err != nil {
		id, err := sip.UACReadRequestDialogID(req)
		if err != nil {
			tx.Respond(sip.NewResponseFromRequest(req, sip.StatusBadRequest, "Bad Request", nil))
			return

		}
		// No Error means we have ID
		s, err := DialogsClientCache.DialogLoad(ctx, id)
		if err != nil {
			tx.Respond(sip.NewResponseFromRequest(req, sip.StatusCallTransactionDoesNotExists, "Call/Transaction Does Not Exist", nil))
			return
		}

		s.handleReInviteWithEngine(req, tx)
		return
	}

	s.handleReInviteWithEngine(req, tx)
}
