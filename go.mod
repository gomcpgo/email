module github.com/prasanthmj/email

go 1.20

require (
	github.com/emersion/go-imap v1.2.1
	github.com/emersion/go-message v0.16.0
	github.com/gomcpgo/mcp v0.0.0
	github.com/jordan-wright/email v4.0.1-0.20210109023952-943e75fe5223+incompatible
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/emersion/go-sasl v0.0.0-20200509203442-7bfe0ed36a21 // indirect
	github.com/emersion/go-textwrapper v0.0.0-20200911093747-65d896831594 // indirect
	golang.org/x/text v0.14.0 // indirect
)

replace github.com/gomcpgo/mcp => ../mcp
