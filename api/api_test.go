package api

import (
	"context"
	"encoding/json"
	"math"
	"net/http/httptest"
	"os"
	"testing"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/accesstoken"
	"github.com/bytom/blockchain/rpc"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/consensus"
	"github.com/bytom/testutil"
)

func TestAPIHandler(t *testing.T) {
	a := &API{}
	response := &Response{}

	// init httptest server
	a.buildHandler()
	server := httptest.NewServer(a.handler)
	defer server.Close()

	// create accessTokens
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")
	a.accessTokens = accesstoken.NewStore(testDB)

	client := &rpc.Client{
		BaseURL:     server.URL,
		AccessToken: "test-user:test-secret",
	}

	cases := []struct {
		path     string
		request  interface{}
		respWant *Response
	}{
		{
			path: "/create-key",
			request: struct {
				Alias    string `json:"alias"`
				Password string `json:"password"`
			}{Alias: "alice", Password: "123456"},
			respWant: &Response{
				Status: "fail",
				Msg:    "wallet not found, please check that the wallet is open",
			},
		},
		{
			path:    "/error",
			request: nil,
			respWant: &Response{
				Status: "fail",
				Msg:    "wallet not found, please check that the wallet is open",
			},
		},
		{
			path:    "/",
			request: nil,
			respWant: &Response{
				Status: "",
				Msg:    "",
			},
		},
		{
			path: "/create-access-token",
			request: struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			}{ID: "test-access-id", Type: "test-access-type"},
			respWant: &Response{
				Status: "success",
				Msg:    "",
				Data:   map[string]interface{}{"id": "test-access-id", "type": "test-access-type", "token": "test-access-id:440d87ae0d625a7fcf076275b18372e09a0899e37ec86398879388de90cb0c67"},
			},
		},
		{
			path:    "/gas-rate",
			request: nil,
			respWant: &Response{
				Status: "success",
				Msg:    "",
				Data:   map[string]interface{}{"gasRate": 1000},
			},
		},
	}

	for _, c := range cases {
		response = &Response{}
		client.Call(context.Background(), c.path, c.request, &response)

		if !testutil.DeepEqual(response.Status, c.respWant.Status) {
			t.Errorf(`got=%#v; want=%#v`, response.Status, c.respWant.Status)
		}
	}
}

func TestEstimateTxGas(t *testing.T) {
	tmplStr := `{"allow_additional_actions":false,"raw_transaction":"070100010161015ffe8a1209937a6a8b22e8c01f056fd5f1730734ba8964d6b79de4a639032cecddffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8099c4d59901000116001485eb6eee8023332da85df60157dc9b16cc553fb2010002013dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80afa08b4f011600142b4fd033bc76b4ddf5cb00f625362c4bc7b10efa00013dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8090dfc04a011600146eea1ce6cfa5b718ae8094376be9bc1a87c9c82700","signing_instructions":[{"position":0,"witness_components":[{"keys":[{"derivation_path":["010100000000000000","0100000000000000"],"xpub":"cb4e5932d808ee060df9552963d87f60edac42360b11d4ad89558ef2acea4d4aaf4818f2ebf5a599382b8dfce0a0c798c7e44ec2667b3a1d34c61ba57609de55"}],"quorum":1,"signatures":null,"type":"raw_tx_signature"},{"type":"data","value":"1c9b5c1db7f4afe31fd1b7e0495a8bb042a271d8d7924d4fc1ff7cf1bff15813"}]}]}`
	template := txbuilder.Template{}
	err := json.Unmarshal([]byte(tmplStr), &template)
	if err != nil {
		t.Fatal(err)
	}

	estimateResult, err := EstimateTxGas(template)
	if err != nil {
		t.Fatal(err)
	}

	baseRate := float64(100000)
	totalNeu := float64(estimateResult.StorageNeu+estimateResult.VMNeu+flexibleGas*consensus.VMGasRate) / baseRate
	roundingNeu := math.Ceil(totalNeu)
	estimateNeu := int64(roundingNeu) * int64(baseRate)

	if estimateResult.TotalNeu != estimateNeu {
		t.Errorf(`got=%#v; want=%#v`, estimateResult.TotalNeu, estimateNeu)
	}
}

func TestEstimateTxGasRange(t *testing.T) {

	cases := []struct {
		path     string
		tmplStr  string
		respWant *EstimateTxGasResp
	}{
		{
			path:    "/estimate-transaction-gas",
			tmplStr: `{"raw_transaction":"070100010160015e9a4e2bbae57dd71b6a827fb50aaeb744ce3ae6f45c4aec7494ad097213220e8affffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0cea1bc5800011600144a6322008c5424251c7502c7d7d55f6389c3c358010001013dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe086f29b3301160014fa61b0629e5f2da2bb8b08e7fc948dbd265234f700","signing_instructions":[{"position":0,"witness_components":[{"type":"raw_tx_signature","quorum":1,"keys":[{"xpub":"19204fe9172cb0eeae86b39ec7a61ddc556656c8df08fd43ef6074296f32b347349722316972e382c339b79b7e1d83a565c6b3e7cf46847733a47044ae493257","derivation_path":["010100000000000000","0700000000000000"]}],"signatures":null},{"type":"data","value":"a527a92a7488c010bc42b39d6b50f0822183e51efab228af8ca8ca81ca459237"}]}],"allow_additional_actions":false}`,
			respWant: &EstimateTxGasResp{
				TotalNeu: (flexibleGas + 2095) * consensus.VMGasRate,
			},
		},
		{
			path:    "/estimate-transaction-gas",
			tmplStr: `{"raw_transaction":"07010001016d016bcf24f1471d67c25a01ac84482ecdd8550229180171cae22321f87fe43d4f6a13ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8086a6d5f2020001220020713ef71e6087a58d6055ce81e8a8ea8a60ca19aef77923859e53a1fa9df0042989010240844b99bab9f393e89ca3bb272b1ba146852124f13a2d37fc47da6a7320f5ae1a4b6df1322750906ad480796db663e35ef7fd9544718eea08e51c5388f9813d0446ae20bd609e953918ab2ce120c43486894ff38dc4b65c2c1b4e19f6b41265d76b062120508684f922c1e5eea3dcbd592b00d297b2ddf92d35d5acabea9ff491ef514abe5152ad02014affffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80bef6b4cd0201220020dc794f041d19c67108a05d2a6d797a2b12029f31b2c91ec699c9477727f25315000149ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80fef9b123012200200824e931fb806bd77fdcd291aad3bd0a4493443a4120062bd659e64a3e0bac6600","signing_instructions":[{"position":0,"witness_components":[{"type":"raw_tx_signature","quorum":1,"keys":[{"xpub":"5ff7f79f0fd4eb9ccb17191b0a1ac9bed5b4a03320a06d2ff8170dd51f9ad9089c4038ec7280b5eb6745ef3d36284e67f5cf2ed2a0177d462d24abf53c0399ed","derivationPath":["010200000000000000","0400000000000000"]},{"xpub":"d0e7607bec7f68ea9135fbb9e3e94ef05a034d28be847070740fcba9454a749f6e21942cfef90f1437184cb70775beb290c13852c1497631dbcb137f74788e4f","derivationPath":["010200000000000000","0400000000000000"]}],"signatures":["","844b99bab9f393e89ca3bb272b1ba146852124f13a2d37fc47da6a7320f5ae1a4b6df1322750906ad480796db663e35ef7fd9544718eea08e51c5388f9813d04"]},{"type":"data","value":"ae20bd609e953918ab2ce120c43486894ff38dc4b65c2c1b4e19f6b41265d76b062120508684f922c1e5eea3dcbd592b00d297b2ddf92d35d5acabea9ff491ef514abe5152ad"}]}],"allow_additional_actions":false}`,
			respWant: &EstimateTxGasResp{
				TotalNeu: (flexibleGas + 3305) * consensus.VMGasRate,
			},
		},
		{
			path:    "/estimate-transaction-gas",
			tmplStr: `{"raw_transaction":"07010002016c016acf24f1471d67c25a01ac84482ecdd8550229180171cae22321f87fe43d4f6a13ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80b4c4c32101012200200824e931fb806bd77fdcd291aad3bd0a4493443a4120062bd659e64a3e0bac66ef020440f5baa1530bd7ded5c37f1c91360e28e736c91a7933eff961d68eebf90bdce63eb4361689759a8aa420256af565e38921985026de8d27dd7b66f0d01c90170a0440b23b44f62f3e97bcbd5f80cb9bb3d63cb154c62d402851e5b4d5d89849fef74271c8c38f594b944b75222d06ef18bddec4b6278ad3185f72ac5321ce5948e90940a00b096eef5b3bed5c6a2843d29e1820ef1413947d3e278c21cc70976c47976d1159468f071bf853b244be8f6cc55d78615ea6594c946f1a6e6622d8e9d42206a901ae20d441b6f375659325a04eede4fc3b74579bb08ccd05b41b99776501e22d6dca7320af6d98ca2c3cd10bf0affbfa6e86609b750523cfadb662ec963c164f05798a49209820b9f1553b03aaebe7e3f9e9222ed7db73b5079b18564042fd3b2cef74156a20271b52de5f554aa4a6f1358f1c2193617bfb3fed4546d13c4af773096a429f9420eeb4a78d8b5cb8283c221ca2d3fd96b8946b3cddee02b7ceffb8f605932588595355ad016c016a158f56c5673a52876bbbed4cd8724428b43a8d9ddd2a759c9df06b46898f101affffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80fef9b12301012200200824e931fb806bd77fdcd291aad3bd0a4493443a4120062bd659e64a3e0bac66ef020440c3c4fdbe99f9266a42df767cf03c22d9d09096446a8882b9d0c0076d9c85da28add31320705452fb566a091515cedb1ea9966647201236a0da13a020f848b8084043e22fe631cee95e3185ecd0c6fc4a262689d674725abe7d7f3158d8d43c776338edeec76600776fc0dcee280bd7a1a8a2b23909c6cefa7fbb55c27522b6100640fefe403941035a66ba9b6d097dfe0ada68ae6d006272928fad2ba23341fe878690e9e2fa1d2d3992c16aa20125fb2da7f7687920c12a36e4964533ceeccd3602a901ae20d441b6f375659325a04eede4fc3b74579bb08ccd05b41b99776501e22d6dca7320af6d98ca2c3cd10bf0affbfa6e86609b750523cfadb662ec963c164f05798a49209820b9f1553b03aaebe7e3f9e9222ed7db73b5079b18564042fd3b2cef74156a20271b52de5f554aa4a6f1358f1c2193617bfb3fed4546d13c4af773096a429f9420eeb4a78d8b5cb8283c221ca2d3fd96b8946b3cddee02b7ceffb8f605932588595355ad020149ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80ea8ed51f01220020036f3d1665dc802fd36aded656c2f4b2b2c5b00e86c44f5352257b718941a4e9000149ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80fef9b12301220020e402787b2bf9749f8fcdcc132a44e86bacf36780ec5df2189a11020d590533ee00","signing_instructions":[{"position":0,"witness_components":[{"type":"raw_tx_signature","quorum":3,"keys":[{"xpub":"5ff7f79f0fd4eb9ccb17191b0a1ac9bed5b4a03320a06d2ff8170dd51f9ad9089c4038ec7280b5eb6745ef3d36284e67f5cf2ed2a0177d462d24abf53c0399ed","derivationPath":["010300000000000000","0100000000000000"]},{"xpub":"7d1c7a9094ab23f432e60afbbfe2791ba9ab3daba8aaa544634218243b8659985cb0ae9fe2b0f5da8a84c6b117c9491bf38f5e59b0d05642d90ba34cf7611eec","derivationPath":["010300000000000000","0100000000000000"]},{"xpub":"b0d2d90cdee01976d51b55963ae214493708d8db44f7516d2d4853a542cba4c07fbd0ad3e7a9ff4b6fbe6b71e66f4538a9424eaf15f538d958aa7025f5f752dc","derivationPath":["010300000000000000","0100000000000000"]},{"xpub":"d0e7607bec7f68ea9135fbb9e3e94ef05a034d28be847070740fcba9454a749f6e21942cfef90f1437184cb70775beb290c13852c1497631dbcb137f74788e4f","derivationPath":["010300000000000000","0100000000000000"]},{"xpub":"e18b9d219e960d761e8d03290acddb5211fea1140c87663908ea74f212763ca8d809bb0fe861884e662429564fa0f2725b5787175054c17685a83a68e160344d","derivationPath":["010300000000000000","0100000000000000"]}],"signatures":["","f5baa1530bd7ded5c37f1c91360e28e736c91a7933eff961d68eebf90bdce63eb4361689759a8aa420256af565e38921985026de8d27dd7b66f0d01c90170a04","b23b44f62f3e97bcbd5f80cb9bb3d63cb154c62d402851e5b4d5d89849fef74271c8c38f594b944b75222d06ef18bddec4b6278ad3185f72ac5321ce5948e909","a00b096eef5b3bed5c6a2843d29e1820ef1413947d3e278c21cc70976c47976d1159468f071bf853b244be8f6cc55d78615ea6594c946f1a6e6622d8e9d42206",""]},{"type":"data","value":"ae20d441b6f375659325a04eede4fc3b74579bb08ccd05b41b99776501e22d6dca7320af6d98ca2c3cd10bf0affbfa6e86609b750523cfadb662ec963c164f05798a49209820b9f1553b03aaebe7e3f9e9222ed7db73b5079b18564042fd3b2cef74156a20271b52de5f554aa4a6f1358f1c2193617bfb3fed4546d13c4af773096a429f9420eeb4a78d8b5cb8283c221ca2d3fd96b8946b3cddee02b7ceffb8f605932588595355ad"}]},{"position":1,"witness_components":[{"type":"raw_tx_signature","quorum":3,"keys":[{"xpub":"5ff7f79f0fd4eb9ccb17191b0a1ac9bed5b4a03320a06d2ff8170dd51f9ad9089c4038ec7280b5eb6745ef3d36284e67f5cf2ed2a0177d462d24abf53c0399ed","derivationPath":["010300000000000000","0100000000000000"]},{"xpub":"7d1c7a9094ab23f432e60afbbfe2791ba9ab3daba8aaa544634218243b8659985cb0ae9fe2b0f5da8a84c6b117c9491bf38f5e59b0d05642d90ba34cf7611eec","derivationPath":["010300000000000000","0100000000000000"]},{"xpub":"b0d2d90cdee01976d51b55963ae214493708d8db44f7516d2d4853a542cba4c07fbd0ad3e7a9ff4b6fbe6b71e66f4538a9424eaf15f538d958aa7025f5f752dc","derivationPath":["010300000000000000","0100000000000000"]},{"xpub":"d0e7607bec7f68ea9135fbb9e3e94ef05a034d28be847070740fcba9454a749f6e21942cfef90f1437184cb70775beb290c13852c1497631dbcb137f74788e4f","derivationPath":["010300000000000000","0100000000000000"]},{"xpub":"e18b9d219e960d761e8d03290acddb5211fea1140c87663908ea74f212763ca8d809bb0fe861884e662429564fa0f2725b5787175054c17685a83a68e160344d","derivationPath":["010300000000000000","0100000000000000"]}],"signatures":["","c3c4fdbe99f9266a42df767cf03c22d9d09096446a8882b9d0c0076d9c85da28add31320705452fb566a091515cedb1ea9966647201236a0da13a020f848b808","43e22fe631cee95e3185ecd0c6fc4a262689d674725abe7d7f3158d8d43c776338edeec76600776fc0dcee280bd7a1a8a2b23909c6cefa7fbb55c27522b61006","fefe403941035a66ba9b6d097dfe0ada68ae6d006272928fad2ba23341fe878690e9e2fa1d2d3992c16aa20125fb2da7f7687920c12a36e4964533ceeccd3602",""]},{"type":"data","value":"ae20d441b6f375659325a04eede4fc3b74579bb08ccd05b41b99776501e22d6dca7320af6d98ca2c3cd10bf0affbfa6e86609b750523cfadb662ec963c164f05798a49209820b9f1553b03aaebe7e3f9e9222ed7db73b5079b18564042fd3b2cef74156a20271b52de5f554aa4a6f1358f1c2193617bfb3fed4546d13c4af773096a429f9420eeb4a78d8b5cb8283c221ca2d3fd96b8946b3cddee02b7ceffb8f605932588595355ad"}]}],"allow_additional_actions":false}`,
			respWant: &EstimateTxGasResp{
				TotalNeu: (flexibleGas + 13556) * consensus.VMGasRate,
			},
		},
	}
	for _, c := range cases {
		template := txbuilder.Template{}
		err := json.Unmarshal([]byte(c.tmplStr), &template)
		if err != nil {
			t.Fatal(err)
		}
		estimateTxGasResp, err := EstimateTxGas(template)
		realTotalNeu := float64(c.respWant.TotalNeu)
		rate := math.Abs((float64(estimateTxGasResp.TotalNeu) - realTotalNeu) / float64(estimateTxGasResp.TotalNeu))
		if rate > 0.2 {
			t.Errorf(`the estimateNeu over realNeu 20%%`)
		}
	}
}
