package server

type (
	NetStat struct {
		recievedBytes    int
		sentBytes        int
		recievedMessages int
		sentMessages     int
		clientErrors     int
		serverErrors     int
	}
)

func (ns *NetStat) RecievedBytes() int {
	return ns.recievedBytes
}

func (ns *NetStat) SentBytes() int {
	return ns.sentBytes
}

func (ns *NetStat) ClientErrors() int {
	return ns.clientErrors
}

func (ns *NetStat) ServerErrors() int {
	return ns.serverErrors
}

func (ns *NetStat) AddClientErrors(n int) {
	ns.clientErrors += n
}

func (ns *NetStat) AddServerErrors(n int) {
	ns.serverErrors += n
}

func (ns *NetStat) AddRecievedBytes(n int) {
	ns.recievedBytes += n
}

func (ns *NetStat) AddSentBytes(n int) {
	ns.sentBytes += n
}
