module github.com/bytom/bytom

go 1.16

replace (
    github.com/agl/ed25519 => ./lib/github.com/tendermint/ed25519
	github.com/golang/protobuf => github.com/golang/protobuf v1.0.0
	github.com/prometheus/prometheus/util/flock => ./lib/github.com/prometheus/prometheus/util/flock
	github.com/tendermint/go-wire => github.com/tendermint/go-amino v0.6.2
	golang.org/x/crypto => ./lib/golang.org/x/crypto
	golang.org/x/net => ./lib/golang.org/x/net
	gonum.org/v1/gonum/mat => github.com/gonum/gonum/mat v0.9.1
	google.golang.org/grpc => ./lib/google.golang.org/grpc
)

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/agl/ed25519 v0.0.0-00010101000000-000000000000 // indirect
	github.com/btcsuite/go-socks v0.0.0-20170105172521-4720035b7bfd
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/cespare/cp v1.1.1
	github.com/cpuguy83/go-md2man v1.0.10
	github.com/davecgh/go-spew v1.1.1
	github.com/fastly/go-utils v0.0.0-20180712184237-d95a45783239
	github.com/fsnotify/fsnotify v1.4.9
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da
	github.com/golang/protobuf v1.5.2
	github.com/golang/snappy v0.0.3
	github.com/gorilla/websocket v1.4.2
	github.com/grandcat/zeroconf v1.0.0
	github.com/hashicorp/go-version v1.3.0
	github.com/hashicorp/hcl v1.0.0
	github.com/holiman/uint256 v1.1.1
	github.com/inconshreveable/mousetrap v1.0.0
	github.com/jehiah/go-strftime v0.0.0-20171201141054-1d33003b3869
	github.com/johngb/langreg v0.0.0-20150123211413-5c6abc6d19d2
	github.com/jonboulle/clockwork v0.2.2
	github.com/kr/secureheader v0.2.0
	github.com/lestrrat-go/envload v0.0.0-20180220234015-a3eb8ddeffcc
	github.com/lestrrat-go/file-rotatelogs v2.4.0+incompatible
	github.com/lestrrat-go/strftime v1.0.4
	github.com/magiconair/properties v1.8.5
	github.com/miekg/dns v1.1.41
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.4.1
	github.com/onsi/ginkgo v1.16.1
	github.com/onsi/gomega v1.11.0
	github.com/pborman/uuid v1.2.1
	github.com/pelletier/go-toml v1.9.0
	github.com/pkg/errors v0.9.1
	github.com/pkg/sftp v1.13.0
	github.com/pmezard/go-difflib v1.0.0
	github.com/prometheus/prometheus/util/flock v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.8.1
	github.com/smartystreets/goconvey v1.6.4
	github.com/spf13/afero v1.6.0
	github.com/spf13/cast v1.3.1
	github.com/spf13/cobra v1.1.3
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	github.com/stretchr/objx v0.3.0
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20200815110645-5c35d600f0ca
	github.com/tebeka/strftime v0.1.5
	github.com/tendermint/ed25519 v0.0.0-20171027050219-d8387025d2b9
	github.com/tendermint/go-amino v0.16.0 // indirect
	github.com/tendermint/go-crypto v0.9.0
	github.com/tendermint/go-wire v0.0.0-00010101000000-000000000000
	github.com/tendermint/tendermint v0.34.8 // indirect
	github.com/tendermint/tmlibs v0.9.0
	github.com/toqueteos/webbrowser v1.2.0
	github.com/xordataexchange/crypt v0.0.3-0.20170626215501-b2862e3d0a77
	github.com/zondax/ledger-go v0.12.1 // indirect
	github.com/zondax/ledger-goclient v0.9.9 // indirect
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	golang.org/x/exp v0.0.0-20210405174845-4513512abef3
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4
	golang.org/x/sys v0.0.0-20210403161142-5e06dd20ab57
	golang.org/x/text v0.3.6
	golang.org/x/tools v0.1.0
	gonum.org/v1/gonum v0.9.1
	google.golang.org/genproto v0.0.0-20210406143921-e86de6bf7a46
	google.golang.org/grpc v1.36.1
	gopkg.in/airbrake/gobrake.v2 v2.0.9 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c
	gopkg.in/fatih/set.v0 v0.2.1
	gopkg.in/gemnasium/logrus-airbrake-hook.v2 v2.1.2
	gopkg.in/karalabe/cookiejar.v2 v2.0.0-20150724131613-8dcd6a7f4951
	gopkg.in/yaml.v2 v2.4.0
)
