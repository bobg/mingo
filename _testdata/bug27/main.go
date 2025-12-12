package main

import "crypto/tls"

func main() {
	var foo tls.QUICSessionTicketOptions
	print(foo.Extra)
}
