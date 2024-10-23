package rtpengine

import (
	"github.com/SilvaMendes/go-rtpengine"
	"github.com/rs/zerolog"
)

type RTP struct {
	*rtpengine.Client
	*rtpengine.Engine
	log zerolog.Logger

	conf engineClient
}

type EngineOption func(engine engineClient)

type engineClient struct {
	Proto  string
	Port   int
	Ip     string
	Domain string
}

func WithProto(proto string) EngineOption {
	return func(e engineClient) {
		e.Proto = proto
	}
}

func WithIp(ip string) EngineOption {
	return func(e engineClient) {
		e.Ip = ip
	}
}

func WithPort(port int) EngineOption {
	return func(e engineClient) {
		e.Port = port
	}
}

func WithDomain(domain string) EngineOption {
	return func(e engineClient) {
		e.Domain = domain
	}
}

// Create new rtpengine instance
func NewEngine(opts ...EngineOption) *RTP {
	rtp := &RTP{}
	for _, o := range opts {
		o(rtp.conf)
	}

	if rtp.conf.Port == 0 {
		rtp.conf.Port = 22222
	}

	if rtp.conf.Proto == "" {
		rtp.Client, _ = rtpengine.NewClient(rtp.Engine, rtpengine.WithClientPort(rtp.conf.Port), rtpengine.WithClientProto("udp"), rtpengine.WithClientIP(rtp.conf.Ip))
		rtp.log.Warn().Msg("Empty protocol, by default UDP added")
	}

	if rtp.conf.Ip != "" && rtp.conf.Domain == "" {
		rtp.Client, _ = rtpengine.NewClient(rtp.Engine, rtpengine.WithClientPort(rtp.conf.Port), rtpengine.WithClientProto(rtp.conf.Proto), rtpengine.WithClientIP(rtp.conf.Ip))
	}

	if rtp.conf.Ip == "" && rtp.conf.Domain != "" {
		rtp.Client, _ = rtpengine.NewClient(rtp.Engine, rtpengine.WithClientPort(rtp.conf.Port), rtpengine.WithClientProto(rtp.conf.Proto), rtpengine.WithClientDns(rtp.conf.Domain))
	}

	if rtp.conf.Ip != "" && rtp.conf.Domain != "" {
		rtp.Client, _ = rtpengine.NewClient(rtp.Engine, rtpengine.WithClientPort(rtp.conf.Port), rtpengine.WithClientProto(rtp.conf.Proto), rtpengine.WithClientIP(rtp.conf.Ip))
		rtp.log.Warn().Msg("Domain and IP cannot be configured at the same time by default it will be the IP")
	}

	return rtp
}

// Profile to offer a UDP sdp protocol
func (r *RTP) RTP_UDP_Offer(params *rtpengine.ParamsOptString) *rtpengine.RequestRtp {
	rtcpmux := make([]rtpengine.ParamRTCPMux, 0)
	replace := make([]rtpengine.ParamReplace, 0)
	flags := make([]rtpengine.ParamFlags, 0)
	sdes := make([]rtpengine.SDES, 0)

	rtcpmux = append(rtcpmux, rtpengine.RTCPDemux)
	replace = append(replace, rtpengine.SessionName, rtpengine.Origin)
	flags = append(flags, rtpengine.StripExtmap, rtpengine.NoRtcpAttribute)
	sdes = append(sdes, rtpengine.SDESOff)

	// Set protocol to RTP/AVP, set ICE and DTLS
	params.TransportProtocol = rtpengine.RTP_AVP
	params.ICE = rtpengine.ICERemove
	params.DTLS = rtpengine.DTLSOff

	r.log.Info().Str("Offer", string(rtpengine.RTP_AVP)).Msg("Profile to offer a UDP sdp protocol")

	return &rtpengine.RequestRtp{
		Command:         string(rtpengine.Offer),
		ParamsOptString: params,
		ParamsOptInt:    &rtpengine.ParamsOptInt{},
		ParamsOptStringArray: &rtpengine.ParamsOptStringArray{
			Flags:   flags,
			RtcpMux: rtcpmux,
			SDES:    sdes,
			Replace: replace,
		},
	}
}

// Profile to offer a TCP sdp protocol
func (r *RTP) RTP_TCP_Offer(params *rtpengine.ParamsOptString) *rtpengine.RequestRtp {
	rtcpmux := make([]rtpengine.ParamRTCPMux, 0)
	replace := make([]rtpengine.ParamReplace, 0)
	flags := make([]rtpengine.ParamFlags, 0)
	osrtp := make([]rtpengine.OSRTP, 0)

	rtcpmux = append(rtcpmux, rtpengine.RTCPDemux)
	replace = append(replace, rtpengine.SessionName, rtpengine.Origin)
	flags = append(flags, rtpengine.LoopProtect, rtpengine.StrictSource)
	osrtp = append(osrtp, rtpengine.OSRTPOffer)

	// Set protocol to RTP/AVP, set ICE and DTLS
	params.TransportProtocol = rtpengine.RTP_AVP
	params.ICE = rtpengine.ICERemove
	params.DTLS = rtpengine.DTLSOff

	r.log.Info().Str("Offer", string(rtpengine.RTP_AVP)).Msg("Profile to offer a TCP sdp protocol")

	return &rtpengine.RequestRtp{
		Command:         string(rtpengine.Offer),
		ParamsOptString: params,
		ParamsOptInt:    &rtpengine.ParamsOptInt{},
		ParamsOptStringArray: &rtpengine.ParamsOptStringArray{
			Flags:   flags,
			RtcpMux: rtcpmux,
			OSRTP:   osrtp,
			Replace: replace,
		},
	}
}

// Profile to offer a TLS sdp protocol
func (r *RTP) RTP_TLS_Offer(params *rtpengine.ParamsOptString) *rtpengine.RequestRtp {
	rtcpmux := make([]rtpengine.ParamRTCPMux, 0)
	replace := make([]rtpengine.ParamReplace, 0)
	flags := make([]rtpengine.ParamFlags, 0)
	osrtp := make([]rtpengine.OSRTP, 0)

	rtcpmux = append(rtcpmux, rtpengine.RTCPOffer)

	replace = append(replace, rtpengine.SessionName, rtpengine.Origin)
	flags = append(flags, rtpengine.LoopProtect, rtpengine.TrustAddress)
	osrtp = append(osrtp, rtpengine.OSRTPAccept)

	// Set protocol to RTP/SAVP, set ICE and DTLS
	params.TransportProtocol = rtpengine.RTP_SAVP
	params.ICE = rtpengine.ICERemove
	params.DTLS = rtpengine.DTLSOff

	r.log.Info().Str("Offer", string(rtpengine.RTP_SAVP)).Msg("Profile to offer a TLS sdp protocol")

	return &rtpengine.RequestRtp{
		Command:         string(rtpengine.Offer),
		ParamsOptString: params,
		ParamsOptInt:    &rtpengine.ParamsOptInt{},
		ParamsOptStringArray: &rtpengine.ParamsOptStringArray{
			Flags:   flags,
			RtcpMux: rtcpmux,
			OSRTP:   osrtp,
			Replace: replace,
		},
	}
}

// Profile to offer a WS sdp protocol
func (r *RTP) RTP_WS_Offer(params *rtpengine.ParamsOptString) *rtpengine.RequestRtp {
	rtcpmux := make([]rtpengine.ParamRTCPMux, 0)
	replace := make([]rtpengine.ParamReplace, 0)
	flags := make([]rtpengine.ParamFlags, 0)
	sdes := make([]rtpengine.SDES, 0)

	rtcpmux = append(rtcpmux, rtpengine.RTCPOffer)
	replace = append(replace, rtpengine.SessionName, rtpengine.Origin)
	flags = append(flags, rtpengine.LoopProtect)
	sdes = append(sdes, rtpengine.SDESPad)

	// Set protocol to UDP/TLS/RTP/SAVP, set ICE and DTLS
	params.TransportProtocol = rtpengine.UDP_TLS_RTP_SAVP
	params.ICE = rtpengine.ICEForce
	params.DTLS = rtpengine.DTLSPassive

	r.log.Info().Str("Offer", string(rtpengine.UDP_TLS_RTP_SAVP)).Msg("Profile to offer a WS sdp protocol")

	return &rtpengine.RequestRtp{
		Command:         string(rtpengine.Offer),
		ParamsOptString: params,
		ParamsOptInt:    &rtpengine.ParamsOptInt{},
		ParamsOptStringArray: &rtpengine.ParamsOptStringArray{
			Flags:   flags,
			RtcpMux: rtcpmux,
			SDES:    sdes,
			Replace: replace,
		},
	}
}

// Profile to offer a WSS sdp protocol
func (r *RTP) RTP_WSS_Offer(params *rtpengine.ParamsOptString) *rtpengine.RequestRtp {
	rtcpmux := make([]rtpengine.ParamRTCPMux, 0)
	replace := make([]rtpengine.ParamReplace, 0)
	flags := make([]rtpengine.ParamFlags, 0)
	sdes := make([]rtpengine.SDES, 0)

	rtcpmux = append(rtcpmux, rtpengine.RTCPOffer)
	replace = append(replace, rtpengine.SessionName, rtpengine.Origin)
	flags = append(flags, rtpengine.LoopProtect, rtpengine.TrickleICE, rtpengine.TrustAddress, rtpengine.StrictSource, rtpengine.Unidirectional)
	sdes = append(sdes, rtpengine.SDESPad)

	// Set protocol to UDP/TLS/RTP/SAVPF, set ICE and DTLS
	params.TransportProtocol = rtpengine.UDP_TLS_RTP_SAVPF
	params.ICE = rtpengine.ICEForce
	params.DTLS = rtpengine.DTLSActive

	r.log.Info().Str("Offer", string(rtpengine.UDP_TLS_RTP_SAVPF)).Msg("Profile to offer a WSS sdp protocol")

	return &rtpengine.RequestRtp{
		Command:         string(rtpengine.Offer),
		ParamsOptString: params,
		ParamsOptInt:    &rtpengine.ParamsOptInt{},
		ParamsOptStringArray: &rtpengine.ParamsOptStringArray{
			Flags:   flags,
			RtcpMux: rtcpmux,
			SDES:    sdes,
			Replace: replace,
		},
	}
}

func (r *RTP) RTP_Delete(params *rtpengine.ParamsOptString) *rtpengine.RequestRtp {
	return &rtpengine.RequestRtp{
		Command:              string(rtpengine.Delete),
		ParamsOptString:      params,
		ParamsOptInt:         &rtpengine.ParamsOptInt{},
		ParamsOptStringArray: &rtpengine.ParamsOptStringArray{},
	}
}

// Profile to answer a UDP sdp protocol
func (r *RTP) RTP_UDP_Answer(params *rtpengine.ParamsOptString) *rtpengine.RequestRtp {
	rtcpmux := make([]rtpengine.ParamRTCPMux, 0)
	replace := make([]rtpengine.ParamReplace, 0)
	flags := make([]rtpengine.ParamFlags, 0)
	sdes := make([]rtpengine.SDES, 0)

	rtcpmux = append(rtcpmux, rtpengine.RTCPDemux)
	replace = append(replace, rtpengine.SessionName, rtpengine.Origin)
	flags = append(flags, rtpengine.StripExtmap, rtpengine.NoRtcpAttribute)
	sdes = append(sdes, rtpengine.SDESOff)

	// Set protocol to RTP/AVP, set ICE and DTLS
	params.TransportProtocol = rtpengine.RTP_AVP
	params.ICE = rtpengine.ICERemove
	params.DTLS = rtpengine.DTLSOff

	r.log.Info().Str("Offer", string(rtpengine.RTP_AVP)).Msg("Profile to offer a UDP sdp protocol")

	return &rtpengine.RequestRtp{
		Command:         string(rtpengine.Answer),
		ParamsOptString: params,
		ParamsOptInt:    &rtpengine.ParamsOptInt{},
		ParamsOptStringArray: &rtpengine.ParamsOptStringArray{
			Flags:   flags,
			RtcpMux: rtcpmux,
			SDES:    sdes,
			Replace: replace,
		},
	}

}
