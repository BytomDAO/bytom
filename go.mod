module github.com/bytom/bytom

go 1.16

replace (
	github.com/tendermint/ed25519 => ./lib/github.com/tendermint/ed25519
	github.com/tendermint/go-wire => github.com/tendermint/go-amino v0.6.2
	github.com/zondax/ledger-goclient => github.com/Zondax/ledger-cosmos-go v0.1.0
	golang.org/x/crypto => ./lib/golang.org/x/crypto
	golang.org/x/net => ./lib/golang.org/x/net
	gonum.org/v1/gonum/mat => github.com/gonum/gonum/mat v0.9.1
)

require (
	github.com/btcsuite/btcd v0.21.0-beta // indirect
	github.com/btcsuite/go-socks v0.0.0-20170105172521-4720035b7bfd
	github.com/cespare/cp v1.1.1
	github.com/davecgh/go-spew v1.1.1
	github.com/fortytw2/leaktest v1.3.0 // indirect
	github.com/go-kit/kit v0.10.0 // indirect
	github.com/go-sql-driver/mysql v1.5.0
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da
	github.com/golang/protobuf v1.4.3
	github.com/golang/snappy v0.0.3 // indirect
	github.com/google/uuid v1.2.0
	github.com/gorilla/websocket v1.4.2
	github.com/grandcat/zeroconf v0.0.0-20190424104450-85eadb44205c
	github.com/hashicorp/go-version v1.3.0
	github.com/holiman/uint256 v1.2.0
	github.com/jinzhu/gorm v1.9.16
	github.com/johngb/langreg v0.0.0-20150123211413-5c6abc6d19d2
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/kr/secureheader v0.2.0
	github.com/lestrrat-go/file-rotatelogs v2.4.0+incompatible
	github.com/lestrrat-go/strftime v1.0.4 // indirect
	github.com/magiconair/properties v1.8.5 // indirect
	github.com/miekg/dns v1.1.41 // indirect
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/onsi/ginkgo v1.16.1 // indirect
	github.com/onsi/gomega v1.11.0 // indirect
	github.com/pborman/uuid v1.2.1
	github.com/pelletier/go-toml v1.9.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/prometheus v1.8.2
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/afero v1.6.0 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v1.1.3
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20200815110645-5c35d600f0ca
	github.com/tendermint/ed25519 v0.0.0-20171027050219-d8387025d2b9
	github.com/tendermint/go-crypto v0.2.0
	github.com/tendermint/go-wire v0.16.0
	github.com/tendermint/tmlibs v0.9.0
	github.com/toqueteos/webbrowser v1.2.0
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	golang.org/x/net v0.0.0-20210410081132-afb366fc7cd1 // indirect
	golang.org/x/sys v0.0.0-20210412220455-f1c623a9e750 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/fatih/set.v0 v0.1.0
	gopkg.in/ini.v1 v1.62.0 // indirect
	gopkg.in/karalabe/cookiejar.v2 v2.0.0-20150724131613-8dcd6a7f4951
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)
