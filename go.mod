module github.com/block26/TheAlgoV2

go 1.13

require (
	cloud.google.com/go/firestore v1.0.0 // indirect
	cloud.google.com/go/storage v1.0.0 // indirect
	firebase.google.com/go v3.9.0+incompatible
	github.com/aws/aws-sdk-go v1.24.0
	github.com/block26/exchanges v0.0.0-20190920200622-23118f98fdc4
	github.com/block26/tantra-plot v0.0.0-00010101000000-000000000000
	github.com/gocarina/gocsv v0.0.0-20190821091544-020a928c6f4e
	github.com/stretchr/testify v1.4.0 // indirect
	// gitlab.com/raedah/tradeapi v0.0.0-20191012230305-2229330ff6c7
	go.opencensus.io v0.22.1 // indirect
	golang.org/x/exp v0.0.0-20190912063710-ac5d2bfcbfe0 // indirect
	golang.org/x/sys v0.0.0-20190916202348-b4ddaad3f8a3 // indirect
	golang.org/x/tools v0.0.0-20190917215024-905c8ffbfa41 // indirect
	google.golang.org/api v0.10.0
	google.golang.org/appengine v1.6.2 // indirect
	google.golang.org/genproto v0.0.0-20190916214212-f660b8655731 // indirect
	google.golang.org/grpc v1.23.1 // indirect

)

replace (
	github.com/block26/exchanges => ../../block26/exchanges
	github.com/block26/tantra-plot => ../../block26/tantra-plot
)
