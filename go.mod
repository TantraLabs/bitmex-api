module github.com/tantralabs/TheAlgoV2

go 1.13

// replace github.com/tantralabs/exchanges => ../../tantralabs/exchanges

replace github.com/tantralabs/tradeapi => ../../../github.com/tantralabs/tradeapi

require (
	firebase.google.com/go v3.10.0+incompatible
	github.com/aws/aws-sdk-go v1.25.28
	github.com/c-bata/goptuna v0.1.0
	github.com/fatih/structs v1.1.0
	github.com/gocarina/gocsv v0.0.0-20190927101021-3ecffd272576
	github.com/influxdata/influxdb-client-go v0.1.4
	github.com/influxdata/influxdb1-client v0.0.0-20190809212627-fc22c7df067e
	github.com/jmoiron/sqlx v1.2.0
	github.com/lib/pq v1.2.0
	github.com/tantralabs/exchanges v0.0.0-20191106215748-4d3dd77e096e
	github.com/tantralabs/tradeapi v0.0.0-20191112075701-2692528655ec
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	google.golang.org/api v0.13.0
	gopkg.in/src-d/go-git.v4 v4.8.1
)
