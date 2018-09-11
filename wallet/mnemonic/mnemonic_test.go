package mnemonic

import (
	"crypto/rand"
	"encoding/hex"
	"testing"
)

type vector struct {
	entropy                string
	mnemChineseSimplified  string
	mnemChineseTraditional string
	mnemEnglish            string
	mnemItalian            string
	mnemJapanese           string
	mnemKorean             string
	mnemSpanish            string
	seedChineseSimplified  string
	seedChineseTraditional string
	seedEnglish            string
	seedItalian            string
	seedJapanese           string
	seedKorean             string
	seedSpanish            string
}

func TestNewMnemonic(t *testing.T) {
	for _, vector := range testVectors() {
		testMnem := map[string]string{
			"zh_CN": vector.mnemChineseSimplified,
			"zh_TW": vector.mnemChineseTraditional,
			"en":    vector.mnemEnglish,
			"it":    vector.mnemItalian,
			"ja":    vector.mnemJapanese,
			"ko":    vector.mnemKorean,
			"es":    vector.mnemSpanish,
		}
		testSeed := map[string]string{
			"zh_CN": vector.seedChineseSimplified,
			"zh_TW": vector.seedChineseTraditional,
			"en":    vector.seedEnglish,
			"it":    vector.seedItalian,
			"ja":    vector.seedJapanese,
			"ko":    vector.seedKorean,
			"es":    vector.seedSpanish,
		}
		for key, _ := range wordList {
			entropy, err := hex.DecodeString(vector.entropy)
			assertNil(t, err)

			mnemonic, err := NewMnemonic(entropy, key)
			assertNil(t, err)
			assertEqualString(t, testMnem[key], mnemonic)

			_, err = NewSeedWithErrorChecking(mnemonic, "TREZOR", key)
			assertNil(t, err)

			seed := NewSeed(mnemonic, "TREZOR")
			assertEqualString(t, testSeed[key], hex.EncodeToString(seed))
		}
	}
}

func TestNewMnemonicInvalidEntropy(t *testing.T) {
	_, err := NewMnemonic([]byte{}, "en")
	assertNotNil(t, err)
}

func TestNewSeedWithErrorCheckingInvalidMnemonics(t *testing.T) {
	for _, vector := range badMnemonicSentences() {
		_, err := NewSeedWithErrorChecking(vector.mnemEnglish, "TREZOR", "en")
		assertNotNil(t, err)
	}
}

func TestIsMnemonicValid(t *testing.T) {
	for _, vector := range badMnemonicSentences() {
		assertFalse(t, IsMnemonicValid(vector.mnemEnglish, "en"))
	}

	for _, vector := range testVectors() {
		assertTrue(t, IsMnemonicValid(vector.mnemEnglish, "en"))
	}
}

func TestInvalidMnemonicFails(t *testing.T) {
	for _, vector := range badMnemonicSentences() {
		_, err := MnemonicToByteArray(vector.mnemEnglish, "en")
		assertNotNil(t, err)
	}

	_, err := MnemonicToByteArray("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon yellow", "en")
	assertNotNil(t, err)
	assertEqual(t, err, ErrChecksumIncorrect)
}

func TestNewEntropy(t *testing.T) {
	// Good tests.
	for i := 128; i <= 256; i += 32 {
		_, err := NewEntropy(i)
		assertNil(t, err)
	}
	// Bad Values
	for i := 0; i <= 256; i++ {
		if i%8 != 0 {
			_, err := NewEntropy(i)
			assertNotNil(t, err)
		}
	}
}

func TestMnemonicToByteArrayForDifferentArrayLangths(t *testing.T) {
	max := 1000
	for i := 0; i < max; i++ {
		//16, 20, 24, 28, 32
		length := 16 + (i%5)*4
		seed := make([]byte, length)
		if n, err := rand.Read(seed); err != nil {
			t.Errorf("%v", err)
		} else if n != length {
			t.Errorf("Wrong number of bytes read: %d", n)
		}

		mnemonic, err := NewMnemonic(seed, "en")
		if err != nil {
			t.Errorf("%v", err)
		}

		_, err = MnemonicToByteArray(mnemonic, "en")
		if err != nil {
			t.Errorf("Failed for %x - %v", seed, mnemonic)
		}
	}
}
func TestPadByteSlice(t *testing.T) {
	assertEqualByteSlices(t, []byte{0}, padByteSlice([]byte{}, 1))
	assertEqualByteSlices(t, []byte{0, 1}, padByteSlice([]byte{1}, 2))
	assertEqualByteSlices(t, []byte{1, 1}, padByteSlice([]byte{1, 1}, 2))
	assertEqualByteSlices(t, []byte{1, 1, 1}, padByteSlice([]byte{1, 1, 1}, 2))
}

func TestCompareByteSlices(t *testing.T) {
	assertTrue(t, compareByteSlices([]byte{}, []byte{}))
	assertTrue(t, compareByteSlices([]byte{1}, []byte{1}))
	assertFalse(t, compareByteSlices([]byte{1}, []byte{0}))
	assertFalse(t, compareByteSlices([]byte{1}, []byte{}))
	assertFalse(t, compareByteSlices([]byte{1}, nil))
}

func assertNil(t *testing.T, object interface{}) {
	if object != nil {
		t.Errorf("Expected nil, got %v", object)
	}
}

func assertNotNil(t *testing.T, object interface{}) {
	if object == nil {
		t.Error("Expected not nil")
	}
}

func assertTrue(t *testing.T, a bool) {
	if !a {
		t.Error("Expected true, got false")
	}
}

func assertFalse(t *testing.T, a bool) {
	if a {
		t.Error("Expected false, got true")
	}
}

func assertEqual(t *testing.T, a, b interface{}) {
	if a != b {
		t.Errorf("Objects not equal, expected `%s` and got `%s`", a, b)
	}
}

func assertEqualString(t *testing.T, a, b string) {
	if a != b {
		t.Errorf("Strings not equal, expected `%s` and got `%s`", a, b)
	}
}

func assertEqualByteSlices(t *testing.T, a, b []byte) {
	if len(a) != len(b) {
		t.Errorf("Byte slices not equal, expected %v and got %v", a, b)
		return
	}
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("Byte slices not equal, expected %v and got %v", a, b)
			return
		}
	}
}

func TestMnemonicToByteArrayForZeroLeadingSeeds(t *testing.T) {
	ms := []string{
		"00000000000000000000000000000000",
		"00a84c51041d49acca66e6160c1fa999",
		"00ca45df1673c76537a2020bfed1dafd",
		"0019d5871c7b81fd83d474ef1c1e1dae",
		"00dcb021afb35ffcdd1d032d2056fc86",
		"0062be7bd09a27288b6cf0eb565ec739",
		"00dc705b5efa0adf25b9734226ba60d4",
		"0017747418d54c6003fa64fade83374b",
		"000d44d3ee7c3dfa45e608c65384431b",
		"008241c1ef976b0323061affe5bf24b9",
		"00a6aec77e4d16bea80b50a34991aaba",
		"0011527b8c6ddecb9d0c20beccdeb58d",
		"001c938c503c8f5a2bba2248ff621546",
		"0002f90aaf7a8327698f0031b6317c36",
		"00bff43071ed7e07f77b14f615993bac",
		"00da143e00ef17fc63b6fb22dcc2c326",
		"00ffc6764fb32a354cab1a3ddefb015d",
		"0062ef47e0985e8953f24760b7598cdd",
		"003bf9765064f71d304908d906c065f5",
		"00993851503471439d154b3613947474",
		"007ad0ffe9eae753a483a76af06dfa67",
		"00091824db9ec19e663bee51d64c83cc",
		"00f48ac621f7e3cb39b2012ac3121543",
		"0072917415cdca24dfa66c4a92c885b4",
		"0027ced2b279ea8a91d29364487cdbf4",
		"00b9c0d37fb10ba272e55842ad812583",
		"004b3d0d2b9285946c687a5350479c8c",
		"00c7c12a37d3a7f8c1532b17c89b724c",
		"00f400c5545f06ae17ad00f3041e4e26",
		"001e290be10df4d209f247ac5878662b",
		"00bf0f74568e582a7dd1ee64f792ec8b",
		"00d2e43ecde6b72b847db1539ed89e23",
		"00cecba6678505bb7bfec8ed307251f6",
		"000aeed1a9edcbb4bc88f610d3ce84eb",
		"00d06206aadfc25c2b21805d283f15ae",
		"00a31789a2ab2d54f8fadd5331010287",
		"003493c5f520e8d5c0483e895a121dc9",
		"004706112800b76001ece2e268bc830e",
		"00ab31e28bb5305be56e38337dbfa486",
		"006872fe85df6b0fa945248e6f9379d1",
		"00717e5e375da6934e3cfdf57edaf3bd",
		"007f1b46e7b9c4c76e77c434b9bccd6b",
		"00dc93735aa35def3b9a2ff676560205",
		"002cd5dcd881a49c7b87714c6a570a76",
		"0013b5af9e13fac87e0c505686cfb6bf",
		"007ab1ec9526b0bc04b64ae65fd42631",
		"00abb4e11d8385c1cca905a6a65e9144",
		"00574fc62a0501ad8afada2e246708c3",
		"005207e0a815bb2da6b4c35ec1f2bf52",
		"00f3460f136fb9700080099cbd62bc18",
		"007a591f204c03ca7b93981237112526",
		"00cfe0befd428f8e5f83a5bfc801472e",
		"00987551ac7a879bf0c09b8bc474d9af",
		"00cadd3ce3d78e49fbc933a85682df3f",
		"00bfbf2e346c855ccc360d03281455a1",
		"004cdf55d429d028f715544ce22d4f31",
		"0075c84a7d15e0ac85e1e41025eed23b",
		"00807dddd61f71725d336cab844d2cb5",
		"00422f21b77fe20e367467ed98c18410",
		"00b44d0ac622907119c626c850a462fd",
		"00363f5e7f22fc49f3cd662a28956563",
		"000fe5837e68397bbf58db9f221bdc4e",
		"0056af33835c888ef0c22599686445d3",
		"00790a8647fd3dfb38b7e2b6f578f2c6",
		"00da8d9009675cb7beec930e263014fb",
		"00d4b384540a5bb54aa760edaa4fb2fe",
		"00be9b1479ed680fdd5d91a41eb926d0",
		"009182347502af97077c40a6e74b4b5c",
		"00f5c90ee1c67fa77fd821f8e9fab4f1",
		"005568f9a2dd6b0c0cc2f5ba3d9cac38",
		"008b481f8678577d9cf6aa3f6cd6056b",
		"00c4323ece5e4fe3b6cd4c5c932931af",
		"009791f7550c3798c5a214cb2d0ea773",
		"008a7baab22481f0ad8167dd9f90d55c",
		"00f0e601519aafdc8ff94975e64c946d",
		"0083b61e0daa9219df59d697c270cd31",
	}

	for _, m := range ms {
		seed, _ := hex.DecodeString(m)

		mnemonic, err := NewMnemonic(seed, "en")
		if err != nil {
			t.Errorf("%v", err)
		}

		_, err = MnemonicToByteArray(mnemonic, "en")
		if err != nil {
			t.Errorf("Failed for %x - %v", seed, mnemonic)
		}
	}
}

func badMnemonicSentences() []vector {
	return []vector{
		{mnemEnglish: "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon"},
		{mnemEnglish: "legal winner thank year wave sausage worth useful legal winner thank yellow yellow"},
		{mnemEnglish: "letter advice cage absurd amount doctor acoustic avoid letter advice caged above"},
		{mnemEnglish: "zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo, wrong"},
		{mnemEnglish: "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon"},
		{mnemEnglish: "legal winner thank year wave sausage worth useful legal winner thank year wave sausage worth useful legal will will will"},
		{mnemEnglish: "letter advice cage absurd amount doctor acoustic avoid letter advice cage absurd amount doctor acoustic avoid letter always."},
		{mnemEnglish: "zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo why"},
		{mnemEnglish: "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon art art"},
		{mnemEnglish: "legal winner thank year wave sausage worth useful legal winner thanks year wave worth useful legal winner thank year wave sausage worth title"},
		{mnemEnglish: "letter advice cage absurd amount doctor acoustic avoid letters advice cage absurd amount doctor acoustic avoid letter advice cage absurd amount doctor acoustic bless"},
		{mnemEnglish: "zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo voted"},
		{mnemEnglish: "jello better achieve collect unaware mountain thought cargo oxygen act hood bridge"},
		{mnemEnglish: "renew, stay, biology, evidence, goat, welcome, casual, join, adapt, armor, shuffle, fault, little, machine, walk, stumble, urge, swap"},
		{mnemEnglish: "dignity pass list indicate nasty"},
	}
}

func testVectors() []vector {
	return []vector{
		{
			entropy:                "00000000000000000000000000000000",
			mnemEnglish:            "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about",
			mnemChineseSimplified:  "的 的 的 的 的 的 的 的 的 的 的 在",
			mnemChineseTraditional: "的 的 的 的 的 的 的 的 的 的 的 在",
			mnemItalian:            "abaco abaco abaco abaco abaco abaco abaco abaco abaco abaco abaco abete",
			mnemJapanese:           "あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あおぞら",
			mnemKorean:             "가격 가격 가격 가격 가격 가격 가격 가격 가격 가격 가격 가능",
			mnemSpanish:            "ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco abierto",
			seedEnglish:            "c55257c360c07c72029aebc1b53c05ed0362ada38ead3e3e9efa3708e53495531f09a6987599d18264c1e1c92f2cf141630c7a3c4ab7c81b2f001698e7463b04",
			seedChineseSimplified:  "7f7c7f91ef81f0fb6a3b95b346c50e6472c1d554f8ba90637bad8afce4a4de87c322c1acafa2f6f5e9a8f9b2d2c40e9d389efdc2adbe4445c21a0939fb39e91f",
			seedChineseTraditional: "7f7c7f91ef81f0fb6a3b95b346c50e6472c1d554f8ba90637bad8afce4a4de87c322c1acafa2f6f5e9a8f9b2d2c40e9d389efdc2adbe4445c21a0939fb39e91f",
			seedItalian:            "d2ae4bbd4efc4aba345b66dc2bfa4ea280d85810945ba4e100707694d5731c5a42ac0d0308ba9ad176966879328f1aa014fbcbeb46d671d9475c38254bf1eeb7",
			seedJapanese:           "5a6c23b5abdd5c3e1f7d77ad25ecd715647bdafb44dab324c730a76a45d7421daccee1a4ff0739715a2c56a8a9f1e527a5e3496224d91293bfcd9b5393bfff83",
			seedKorean:             "a253d07f616223e337b6fa257632a2cc37e1ba36ff0bc7cf5a943366fa1b9ef02d6aa0333da51c17902951634b8aa81b6692a194b07f4f8c542335d73c96aad3",
			seedSpanish:            "29a2ee16de47d07025de37e7d9c596869439f9bcd26a702d2bae64db2bf0f68383841c5444b5b3bd39dd720d2ebe59969e110e5955c8e6d32c6c3294fd87439b",
		},
		{
			entropy:                "7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f",
			mnemEnglish:            "legal winner thank year wave sausage worth useful legal winner thank yellow",
			mnemChineseSimplified:  "枪 疫 霉 尝 俩 闹 饿 贤 枪 疫 霉 卿",
			mnemChineseTraditional: "槍 疫 黴 嘗 倆 鬧 餓 賢 槍 疫 黴 卿",
			mnemItalian:            "mimosa vita sussurro zinco vero saltare zattera ulisse mimosa vita sussurro zircone",
			mnemJapanese:           "そつう れきだい ほんやく わかす りくつ ばいか ろせん やちん そつう れきだい ほんやく わかめ",
			mnemKorean:             "실장 활동 큰절 흔적 형제 제대로 훈련 한글 실장 활동 큰절 흔히",
			mnemSpanish:            "ligero vista talar yogur venta queso yacer trozo ligero vista talar zafiro",
			seedEnglish:            "2e8905819b8723fe2c1d161860e5ee1830318dbf49a83bd451cfb8440c28bd6fa457fe1296106559a3c80937a1c1069be3a3a5bd381ee6260e8d9739fce1f607",
			seedChineseSimplified:  "816a69d6866891b246b4d33f54d6d2be624470141754396205d039bdd8003949fec4340253dde4c8e11437a181ad992f56d5b976eb9fbe48f4c5e5fec60a27e1",
			seedChineseTraditional: "f38af46f6bc3222b0f5aa14dd5b8b506e51131510f2450ec9fb52c28617cfa59d436055fe542e25dfa01415639d2171e41796f169f8bbc18516941dfdee8fb72",
			seedItalian:            "f8c609647319a50116e9b7d1a0ec5535c6d08d6c958911fd2c8b2dfd55a61e63e9c6c60c22b5c3aec725acb41980e63cb3ed75fb80648092dee1bbbeab476a6d",
			seedJapanese:           "9d269b22155b3c915b09abfefd4e1104573c528f6977cde89c6a68152c3c714dc6c7e0e62f221c322f3f76e4d0bcca66c06e3d2f6a8d70d612c87dd6dee63976",
			seedKorean:             "e6995bf885f5c64932ca28bbb00bc100a6b89cb6edc987bb05f05f99ae7caf78329029c189834c1cca938000bcf08423da011558a60cf3d90c9035eaaf241b9e",
			seedSpanish:            "1580aa5d5d67057b3a0a12253c283b93921851555529d0bbe9634349d641029216f791ddce3527819d44d833a0df3500b15fd8ba4cae7ca24e1464b9167de633",
		},
		{
			entropy:                "80808080808080808080808080808080",
			mnemEnglish:            "letter advice cage absurd amount doctor acoustic avoid letter advice cage above",
			mnemChineseSimplified:  "壤 对 据 人 三 谈 我 表 壤 对 据 不",
			mnemChineseTraditional: "壤 對 據 人 三 談 我 表 壤 對 據 不",
			mnemItalian:            "misurare afoso bravura accadere alogeno dottore acrilico arazzo misurare afoso bravura abisso",
			mnemJapanese:           "そとづら あまど おおう あこがれる いくぶん けいけん あたえる いよく そとづら あまど おおう あかちゃん",
			mnemKorean:             "실현 감소 기법 가상 걱정 무슨 가족 공간 실현 감소 기법 가득",
			mnemSpanish:            "lino admitir bolero abrir álbum dejar acelga aprender lino admitir bolero abogado",
			seedEnglish:            "d71de856f81a8acc65e6fc851a38d4d7ec216fd0796d0a6827a3ad6ed5511a30fa280f12eb2e47ed2ac03b5c462a0358d18d69fe4f985ec81778c1b370b652a8",
			seedChineseSimplified:  "07b6eada2601141ef9748bdf5af296a134f0f9215a946813b84338dcfba93c8247b0c3429a91e0a1b85a93bd9f1275a9524acecadc9b516c3cf4c8990f44052c",
			seedChineseTraditional: "33f373da1a6b4300dad5cc70d2329ed614512e3c8a423673c294110521326ca66753b9663bdd7c844f17d81609a410a61809dd5113823009f729e2f2f940cab9",
			seedItalian:            "4025269bc4f7550bbc3c61592944946b0d4ac855a5e4582bf86069cc0c9429455cc40d84ba215ed1cec28e27ffc88460c38b9c4e8c486ae878d7c85e95b222bf",
			seedJapanese:           "17914bd3fe4b9e1224c968ec6b967fc6144a5795adbb2636a17f77da9b6b118200ad788672fd06096ca62683940523f5178f6ce3845c967cbd4ad2b3643cc660",
			seedKorean:             "1bb52039a6cc288cf806740836002abce493724edac3d3b9458e3581427df76414b422171ef115d823a01c6b39fa68bd0fed20bf5e64dec008fcb22e4b7f26bb",
			seedSpanish:            "a89366f7f9c4bd98afca8edf1242507506562b8eb8a3a60468cafcb6f3037aba1e4d9a7497f6d49fa94aca87c95703873741441a719325af371f8eda9b59dc83",
		},
		{
			entropy:                "ffffffffffffffffffffffffffffffff",
			mnemEnglish:            "zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo wrong",
			mnemChineseSimplified:  "歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 逻",
			mnemChineseTraditional: "歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 邏",
			mnemItalian:            "zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zerbino",
			mnemJapanese:           "われる われる われる われる われる われる われる われる われる われる われる ろんぶん",
			mnemKorean:             "힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 흑백",
			mnemSpanish:            "zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo yodo",
			seedEnglish:            "ac27495480225222079d7be181583751e86f571027b0497b5b5d11218e0a8a13332572917f0f8e5a589620c6f15b11c61dee327651a14c34e18231052e48c069",
			seedChineseSimplified:  "08ac5d9bed9441013b32bc317aaddeb8310011f219b48239faa4adeeb8b79cb0a3e4d1cb460d2dd37888c0a19bef6edd90ced0fd613d48899eab9ee649d77fcd",
			seedChineseTraditional: "cfd5f4fa6f2a422811951739b1dad9f5291f9cbc977a14ae9dd35dc8ab17aeec9ee6f1455b20f881838f4f945850765dd002a9abcdbe7be002ffcdaf6f63fdaa",
			seedItalian:            "24182cf43f956410b5def9df90e3db0d6f3199c2ebd26e7ddef888ee3bece9101d132e449bb9e1c23dd9ccc6131d2f649c021ee591e88cef8d17cb434ef69efb",
			seedJapanese:           "4bd21b75de4f262b0771a97d6fc877ee19329236ced6e974c4c81a094a5f896758033f7eae270216d727539eee3bc9ba5cad21132a1c6e41a50820e0ac928e83",
			seedKorean:             "b6eb986d6aaf7d0cd0eae2a667ff8bde68c8780fb5a728cf500e29119ce99c9b079a4217836879c1e73b8a85422a85b564d819699a4310a1d007b5be24c24b6d",
			seedSpanish:            "a9d1f751178872cc53fc5433e9b2a97526448adc4b824cedeadd8a127c2416481345dfbef2bfc78275f3498e40b4e8e2e00560100e543aba3f324e752f032bc9",
		},
		{
			entropy:                "000000000000000000000000000000000000000000000000",
			mnemEnglish:            "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon agent",
			mnemChineseSimplified:  "的 的 的 的 的 的 的 的 的 的 的 的 的 的 的 的 的 动",
			mnemChineseTraditional: "的 的 的 的 的 的 的 的 的 的 的 的 的 的 的 的 的 動",
			mnemItalian:            "abaco abaco abaco abaco abaco abaco abaco abaco abaco abaco abaco abaco abaco abaco abaco abaco abaco agitare",
			mnemJapanese:           "あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あらいぐま",
			mnemKorean:             "가격 가격 가격 가격 가격 가격 가격 가격 가격 가격 가격 가격 가격 가격 가격 가격 가격 강도",
			mnemSpanish:            "ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco afición",
			seedEnglish:            "035895f2f481b1b0f01fcf8c289c794660b289981a78f8106447707fdd9666ca06da5a9a565181599b79f53b844d8a71dd9f439c52a3d7b3e8a79c906ac845fa",
			seedChineseSimplified:  "b8fb8047e84951d846dbfbbce3edd0c9e316dc40f35b39f03a837db85f5587ac209088e883b5d924a0a43ad154a636fb65df28fdae821226f0f014a49e773356",
			seedChineseTraditional: "717f4f70c7550da57e42c6b49ac47b5bad3249605ed2f869900596c2de7653a8528380e5c31709ed9c2d19b868bc530158712e97276886b4863d036177bcab33",
			seedItalian:            "2161a4b869f98778b6321714e2502adb11ea120c12163b46fa34e36442ad1981b911a2f9ec82b497e7cd206fa7af2f21a94bb6e4a90159965854784e1558658b",
			seedJapanese:           "a59401a14bb821cce86ec32add8f273a3e07e9c8b1ed430d5d1a06dbf3c083ff2ffb4bb26a384b8faecb58f6cb4c07cfbf2c91108385f6773f2fefd1581926b5",
			seedKorean:             "f40a8db48df9a7fdd73a7b3ceb45f668e4eff098f275a0a5cd739d31572c90aa92bc08b9043d0adf059a945e47e2fdbc26c89dcc15b3893a2a705e4539523ae3",
			seedSpanish:            "6c9f21d46c56f723cd734e308f10ebf44b5b92a2e0d80fd66a2952b8d37af5219e0b93c59e1d8e63b47ac657ec2c524e5fb951d87cac824f84a3ac6264b7aaac",
		},
		{
			entropy:                "7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f",
			mnemEnglish:            "legal winner thank year wave sausage worth useful legal winner thank year wave sausage worth useful legal will",
			mnemChineseSimplified:  "枪 疫 霉 尝 俩 闹 饿 贤 枪 疫 霉 尝 俩 闹 饿 贤 枪 殿",
			mnemChineseTraditional: "槍 疫 黴 嘗 倆 鬧 餓 賢 槍 疫 黴 嘗 倆 鬧 餓 賢 槍 殿",
			mnemItalian:            "mimosa vita sussurro zinco vero saltare zattera ulisse mimosa vita sussurro zinco vero saltare zattera ulisse mimosa virulento",
			mnemJapanese:           "そつう れきだい ほんやく わかす りくつ ばいか ろせん やちん そつう れきだい ほんやく わかす りくつ ばいか ろせん やちん そつう れいぎ",
			mnemKorean:             "실장 활동 큰절 흔적 형제 제대로 훈련 한글 실장 활동 큰절 흔적 형제 제대로 훈련 한글 실장 환갑",
			mnemSpanish:            "ligero vista talar yogur venta queso yacer trozo ligero vista talar yogur venta queso yacer trozo ligero violín",
			seedEnglish:            "f2b94508732bcbacbcc020faefecfc89feafa6649a5491b8c952cede496c214a0c7b3c392d168748f2d4a612bada0753b52a1c7ac53c1e93abd5c6320b9e95dd",
			seedChineseSimplified:  "74187bbdce2dba25eed3b9aebdc65dcb7c61e74c58591451d47f9c7b7b17545a527880640bfb9cab36989eba1edddf57bfce7340697926de7f0b9ec1e0345c38",
			seedChineseTraditional: "2b219a8be0a8e27a6b50d0a74eb42175bd23e22cf4081518c9a74cbfe2cbace46f0adad8d390f8a2ac30feb26226db14fbc545d18ba0e56a853cbf103c92539e",
			seedItalian:            "d9a6205a985fde8c2337f6cc6acf77a93d6ec7dc792551c01400f5d9aaa86aa943416c99fe60be141ca27ab333d9f96648b40b266d6b2d6a6e5b07c8939568be",
			seedJapanese:           "809861f80877e3adc842b0204e401d5aeac1d16d24072f387107f9cf95b639d0a76141ab25d3dc90752472787307a7d8b1a534bea237c2bb348faac973e17488",
			seedKorean:             "3162bc17e0f2f01ee571022444d2c5fbddf6a68dedfe734c319fb574592e9c0328f6526116b3b0b025b23391781d0bef8f43bc8ddc2b054b9f52e1fd6a88e3d2",
			seedSpanish:            "f73b28d7e180e0a92c57276a29489c10a992c8a465ab61be0ade4708543436a682b2a3c22de57c48736ae6f29bebf3e506779c74bc1a835ad6b9f4e174126ca8",
		},
		{
			entropy:                "808080808080808080808080808080808080808080808080",
			mnemEnglish:            "letter advice cage absurd amount doctor acoustic avoid letter advice cage absurd amount doctor acoustic avoid letter always",
			mnemChineseSimplified:  "壤 对 据 人 三 谈 我 表 壤 对 据 人 三 谈 我 表 壤 民",
			mnemChineseTraditional: "壤 對 據 人 三 談 我 表 壤 對 據 人 三 談 我 表 壤 民",
			mnemItalian:            "misurare afoso bravura accadere alogeno dottore acrilico arazzo misurare afoso bravura accadere alogeno dottore acrilico arazzo misurare allievo",
			mnemJapanese:           "そとづら あまど おおう あこがれる いくぶん けいけん あたえる いよく そとづら あまど おおう あこがれる いくぶん けいけん あたえる いよく そとづら いきなり",
			mnemKorean:             "실현 감소 기법 가상 걱정 무슨 가족 공간 실현 감소 기법 가상 걱정 무슨 가족 공간 실현 거액",
			mnemSpanish:            "lino admitir bolero abrir álbum dejar acelga aprender lino admitir bolero abrir álbum dejar acelga aprender lino alacrán",
			seedEnglish:            "107d7c02a5aa6f38c58083ff74f04c607c2d2c0ecc55501dadd72d025b751bc27fe913ffb796f841c49b1d33b610cf0e91d3aa239027f5e99fe4ce9e5088cd65",
			seedChineseSimplified:  "e3629a601f4b87101c4bb36496e3dbd146063351f5e47c048211faddab78efdb91910f0eea5c8e53cfb851aa3e156b0bb5c501b83baaf5f5d4a1679a5bb7d885",
			seedChineseTraditional: "d29225f73231521784d98820ebf0ae4d827c5a9e0c0f8845fd63866cdc70b3a40a2281f3f6c6181c5a53e440528dbf83947a4b2056749cb9cc9c83dcd5c91b0f",
			seedItalian:            "cfb1f800cd5a0f7a8cffb12231fc61739f5f87c963ead5e205dd48221c3417eb1173d3209d9a8ffc4f00ab291bc22c1480b4a0a4fdeef9a1f3916d0ccbed5591",
			seedJapanese:           "01187da93480d0369fff3fc5331284ad6a60cd3ce1f60dbec60899191afa2a2b807cd030038a93ddaf14d4f75d6de4a0e049ee58c92197eb9ca995770b558486",
			seedKorean:             "9fa92e4524e0f7412935b2deea23593c0955f9679d3285e3b955f5cdd2a659ee005ee99bd385f63d82cbdb54a3849229fc9a700e198b65a1452b511884b543eb",
			seedSpanish:            "f799e5c2782b50d0eb1d25b5f94984c5b4037ade236c6aa3b48b3df01b703d8ede5f94555f4e78f87a642a9676ba052865418c469c5739b3e93acc528fad30b7",
		},
		{
			entropy:                "ffffffffffffffffffffffffffffffffffffffffffffffff",
			mnemEnglish:            "zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo when",
			mnemChineseSimplified:  "歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 裕",
			mnemChineseTraditional: "歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 裕",
			mnemItalian:            "zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa vile",
			mnemJapanese:           "われる われる われる われる われる われる われる われる われる われる われる われる われる われる われる われる われる りんご",
			mnemKorean:             "힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 화살",
			mnemSpanish:            "zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo viejo",
			seedEnglish:            "0cd6e5d827bb62eb8fc1e262254223817fd068a74b5b449cc2f667c3f1f985a76379b43348d952e2265b4cd129090758b3e3c2c49103b5051aac2eaeb890a528",
			seedChineseSimplified:  "013c8d6868537176fac7bfa966e6219830008f03b650b0f18a12fd67d9ebf871c400c5f980aa073ddd1b23d60846e357aee193ce7644b574bf65e04cf913e39c",
			seedChineseTraditional: "013c8d6868537176fac7bfa966e6219830008f03b650b0f18a12fd67d9ebf871c400c5f980aa073ddd1b23d60846e357aee193ce7644b574bf65e04cf913e39c",
			seedItalian:            "05a43b9c258f6e83f4073fe4a66d6309e94610fe12dd5d598f4725e4e85ff1fde5ff5b1e61b40e09a481a98953f9dc818342172a460e5e6d17d9ab14874447e2",
			seedJapanese:           "a1385ef66f20a905bbfc70f8be6ecfec341ff76d208e89e1a400ccea34313c99e93f4fba9c6f0729397b9002972af93179dc9dd8af7704fa3d28e656248274dc",
			seedKorean:             "2543a88c8a31570dc9ee868a7b153f7f2e42700778bae7a3aba7017357e708b5cea97e0d9753c9226abc90b83c76ae369d74515ac64102c51a5fd0f809cf8b92",
			seedSpanish:            "2fd3964ac77c52232dc0eb2ab237fea2de9b7509005214101ecbbaeb40f34bce7735e848fca6339f76f289904c6db959fa573fc0aa607d969ac256693b4fb7af",
		},
		{
			entropy:                "0000000000000000000000000000000000000000000000000000000000000000",
			mnemEnglish:            "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon art",
			mnemChineseSimplified:  "的 的 的 的 的 的 的 的 的 的 的 的 的 的 的 的 的 的 的 的 的 的 的 性",
			mnemChineseTraditional: "的 的 的 的 的 的 的 的 的 的 的 的 的 的 的 的 的 的 的 的 的 的 的 性",
			mnemItalian:            "abaco abaco abaco abaco abaco abaco abaco abaco abaco abaco abaco abaco abaco abaco abaco abaco abaco abaco abaco abaco abaco abaco abaco angelo",
			mnemJapanese:           "あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん あいこくしん いってい",
			mnemKorean:             "가격 가격 가격 가격 가격 가격 가격 가격 가격 가격 가격 가격 가격 가격 가격 가격 가격 가격 가격 가격 가격 가격 가격 계단",
			mnemSpanish:            "ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ábaco ancla",
			seedEnglish:            "bda85446c68413707090a52022edd26a1c9462295029f2e60cd7c4f2bbd3097170af7a4d73245cafa9c3cca8d561a7c3de6f5d4a10be8ed2a5e608d68f92fcc8",
			seedChineseSimplified:  "1981c3e3ddfd80f6e9ee1c5ef27ba2697df3d1468496f1d56ae3d8e0b3f0677bbbdfca954e48eb86fe6a36fc0f597bf18ea00248757a01e82182badff94abbbd",
			seedChineseTraditional: "1981c3e3ddfd80f6e9ee1c5ef27ba2697df3d1468496f1d56ae3d8e0b3f0677bbbdfca954e48eb86fe6a36fc0f597bf18ea00248757a01e82182badff94abbbd",
			seedItalian:            "84055239f41c182bbfe6ede6db2e8bc4a97cf86746643b7ea6910c71d67bb2a678a97ecd378cfbf59e30db720b1cfde0faaee73afd3c5deef2188e307d04442c",
			seedJapanese:           "c91afc204a8b098524c5e2134bf4955b9a9ddd5d4bb78c2184bb4378a306e851b60f3e4032fc910ecb48acfb9e441dd3ceaaab9e14700b11396b94e27e8ac2da",
			seedKorean:             "edb71011bc0c227103ba8a769cc36ba609e5407a771727fc0c8cba1b5a44d21ab9163d9deaa37427ccc579864e21f08d0fdd3a53a6be258d3c73b898a01ce2b2",
			seedSpanish:            "f600536eca941ed937318828e9ebab24b3b571558250e7a8342fc3cf16c458b2d7b36c36155a86cc308f7bef6d87b05d5dbe347f1a83c3dfbabd89e9c45b7883",
		},
		{
			entropy:                "7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f",
			mnemEnglish:            "legal winner thank year wave sausage worth useful legal winner thank year wave sausage worth useful legal winner thank year wave sausage worth title",
			mnemChineseSimplified:  "枪 疫 霉 尝 俩 闹 饿 贤 枪 疫 霉 尝 俩 闹 饿 贤 枪 疫 霉 尝 俩 闹 饿 搭",
			mnemChineseTraditional: "槍 疫 黴 嘗 倆 鬧 餓 賢 槍 疫 黴 嘗 倆 鬧 餓 賢 槍 疫 黴 嘗 倆 鬧 餓 搭",
			mnemItalian:            "mimosa vita sussurro zinco vero saltare zattera ulisse mimosa vita sussurro zinco vero saltare zattera ulisse mimosa vita sussurro zinco vero saltare zattera tarpare",
			mnemJapanese:           "そつう れきだい ほんやく わかす りくつ ばいか ろせん やちん そつう れきだい ほんやく わかす りくつ ばいか ろせん やちん そつう れきだい ほんやく わかす りくつ ばいか ろせん まんきつ",
			mnemKorean:             "실장 활동 큰절 흔적 형제 제대로 훈련 한글 실장 활동 큰절 흔적 형제 제대로 훈련 한글 실장 활동 큰절 흔적 형제 제대로 훈련 통로",
			mnemSpanish:            "ligero vista talar yogur venta queso yacer trozo ligero vista talar yogur venta queso yacer trozo ligero vista talar yogur venta queso yacer teatro",
			seedEnglish:            "bc09fca1804f7e69da93c2f2028eb238c227f2e9dda30cd63699232578480a4021b146ad717fbb7e451ce9eb835f43620bf5c514db0f8add49f5d121449d3e87",
			seedChineseSimplified:  "b1eb831927f1c488e233725f9c409dd9bdb9342324393fa56d958e8842623d222510c322f5ba2899428ae08ece8bd87788748c67bdfa73588669ab816c5f3555",
			seedChineseTraditional: "fd50ad67903b2046356e67e55d67309b6f0ccd7c23bfefd049a5b8a40d56c507d73a5517e2d2785f024a7794854594aaad845dd0fbd0432c25a96f2a7181a2cc",
			seedItalian:            "f0e226efcd929216020a9e8f879f06b146d28fecd2856bd401a62ecc0ece8bc6ea717e3f9df523a6a00bd4ca8965e0498d63e779e3156dbf174ebac74ad7be31",
			seedJapanese:           "79aff5bc7868b9054f6c35bb3fa286c72a6931d5999c6c45a029ad31da550b71c8db72e594875e1d61788371b31a03b70fe1d9484840d403e56a1a2783bf9d7e",
			seedKorean:             "dbd640cc9d3e99939bb0fc4473738571e314c29468f01fa85f57e296cf6e8e269d6e32434e46aaa63384930cae83728623195a932a48ccb71a9ea247720d9371",
			seedSpanish:            "3d2a3aec779195f2628e800879d600cfaf2d7fcfa998657068db53906a00608fcc94fc78ceab8c97d6191389c4e468815ea0d11ffa4280c34c3cf17721a27c73",
		},
		{
			entropy:                "8080808080808080808080808080808080808080808080808080808080808080",
			mnemEnglish:            "letter advice cage absurd amount doctor acoustic avoid letter advice cage absurd amount doctor acoustic avoid letter advice cage absurd amount doctor acoustic bless",
			mnemChineseSimplified:  "壤 对 据 人 三 谈 我 表 壤 对 据 人 三 谈 我 表 壤 对 据 人 三 谈 我 五",
			mnemChineseTraditional: "壤 對 據 人 三 談 我 表 壤 對 據 人 三 談 我 表 壤 對 據 人 三 談 我 五",
			mnemItalian:            "misurare afoso bravura accadere alogeno dottore acrilico arazzo misurare afoso bravura accadere alogeno dottore acrilico arazzo misurare afoso bravura accadere alogeno dottore acrilico baco",
			mnemJapanese:           "そとづら あまど おおう あこがれる いくぶん けいけん あたえる いよく そとづら あまど おおう あこがれる いくぶん けいけん あたえる いよく そとづら あまど おおう あこがれる いくぶん けいけん あたえる うめる",
			mnemKorean:             "실현 감소 기법 가상 걱정 무슨 가족 공간 실현 감소 기법 가상 걱정 무슨 가족 공간 실현 감소 기법 가상 걱정 무슨 가족 구속",
			mnemSpanish:            "lino admitir bolero abrir álbum dejar acelga aprender lino admitir bolero abrir álbum dejar acelga aprender lino admitir bolero abrir álbum dejar acelga aumento",
			seedEnglish:            "c0c519bd0e91a2ed54357d9d1ebef6f5af218a153624cf4f2da911a0ed8f7a09e2ef61af0aca007096df430022f7a2b6fb91661a9589097069720d015e4e982f",
			seedChineseSimplified:  "470e61f7e976fa18c7d559e842ba7f39849b2f72ef15428f4276c5160002f36416cd22c2a86bb686d69f6b91818538aa57ae1aab27b3181b92132c59be2b329b",
			seedChineseTraditional: "d029fc9737b801cb4f9aadf5feed02a117b76ead7058e055cc39cb44864023eb492e6a15c68569d6a03a5b11bf15a456c64e1781a553589b47ab569801239a00",
			seedItalian:            "ef549c1e44a7b183031b41f9f692795406de605e43ecc628911a38d7c92f392660c48313a08cf1a055a420d4a8c6b12bef7ff354c903303bc3a5dc12948ff5be",
			seedJapanese:           "0f46c02350b3f1227c3566dea2ff0f2caf716495a95725b320a31a3058d5d62596fdb816be75909d2c5f7094beb171dc504ea8ea60f5e2e40bd8aa0d9339aab0",
			seedKorean:             "9a0ec04a48287ae628d61428f921de5f40fc1035f21883798e05c36f9705b2525a00ebd6bb89fcae9b8af8e9861d0083de331199d6b85b24cff598609a49b305",
			seedSpanish:            "dd095dddb50de059f5cb6932d529ad37dd32d40f72da3d0c7671ffc6bd967b4392fe233e5e9a4d9e5e60413160ae215e34375db85e95ccbab4fd4712f32216ab",
		},
		{
			entropy:                "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			mnemEnglish:            "zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo vote",
			mnemChineseSimplified:  "歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 佳",
			mnemChineseTraditional: "歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 歇 佳",
			mnemItalian:            "zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa zuppa vedetta",
			mnemJapanese:           "われる われる われる われる われる われる われる われる われる われる われる われる われる われる われる われる われる われる われる われる われる われる われる らいう",
			mnemKorean:             "힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 힘껏 허용",
			mnemSpanish:            "zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo zurdo varón",
			seedEnglish:            "dd48c104698c30cfe2b6142103248622fb7bb0ff692eebb00089b32d22484e1613912f0a5b694407be899ffd31ed3992c456cdf60f5d4564b8ba3f05a69890ad",
			seedChineseSimplified:  "8e6607a07fa664d6e4ead23fcc08caf72216d6f078c3b2e5be94e4b6e8d64c784d36bf9b70144fa05840e9a49899128111be5093a2b552b6ab76c0906e9b0e65",
			seedChineseTraditional: "8e6607a07fa664d6e4ead23fcc08caf72216d6f078c3b2e5be94e4b6e8d64c784d36bf9b70144fa05840e9a49899128111be5093a2b552b6ab76c0906e9b0e65",
			seedItalian:            "5089f33aee7852d86a01e8afbfdc8a0ad5af51538e62e3f007d098fa4fc9817ddc990fa87b7235273798e2df52228b62738df923bc2d711fed9cc0558b3ebfec",
			seedJapanese:           "a0705c2feebefb61509dcc49c57586c35379c1981c688fc1d452da44443d9a651a374f1ad2ee3d7847b50655cf9241d7e607be436c0df7c8bac42f2a82985a79",
			seedKorean:             "340bd57209e54e8bde6ca750147933f7e44995047da87b61f64f70f26f289a377e25a65f5efb11f9e651917ec9866d54846516ae0fba956f5f536422bb47d91c",
			seedSpanish:            "deea21c6902df5ef4a8efab8e14de53004c68817ea3de421cdd184f4159a6e9947376ed794c3ce67534f37f80b46674e85335555b5c53f44fdfef27991fedc0e",
		},
		{
			entropy:                "77c2b00716cec7213839159e404db50d",
			mnemEnglish:            "jelly better achieve collect unaware mountain thought cargo oxygen act hood bridge",
			mnemChineseSimplified:  "课 军 个 群 汁 揭 涌 东 滚 他 背 统",
			mnemChineseTraditional: "課 軍 個 群 汁 揭 湧 東 滾 他 背 統",
			mnemItalian:            "malgrado ausilio acqua clinica trincea omissione svista burrasca pervaso adagio istituto bere",
			mnemJapanese:           "せまい うちがわ あずき かろう めずらしい だんち ますく おさめる ていぼう あたる すあな えしゃく",
			mnemKorean:             "시각 교문 가장 달력 하드웨어 연출 태권도 김치 웃음 각자 소용 그룹",
			mnemSpanish:            "jungla asumir acción cedro tóxico mismo tapa brisa obispo ácido hombre baño",
			seedEnglish:            "b5b6d0127db1a9d2226af0c3346031d77af31e918dba64287a1b44b8ebf63cdd52676f672a290aae502472cf2d602c051f3e6f18055e84e4c43897fc4e51a6ff",
			seedChineseSimplified:  "0c510ef7585a9e506ef92152955ecda644398f475dc40ce642e0fabd3cc4dad74d0f42a224c557c66b2d90fef60fd7c58c73fade3ea261c612325c37d7cfe11b",
			seedChineseTraditional: "bf346a4b09f31be3b6d0aa4e840d7d8e6a6420ee50fce7348e7312e89ce4ea8536c2d1b5969d5e9e77f7ff269df126e6edf9d40a937a72799fb31a8ee0860613",
			seedItalian:            "25d048482d5ce15a5b2c412f23e8ae1ea4fbd19bcd5002b5a18bf045ac8ec6fa4ba95c34af1ff667602d28a51906ab7fa0cefc19b67bc2e780dbd21c244857f7",
			seedJapanese:           "b7f5478674839a3487a271014f066059490161a381ec57e9a00de0a3c7311ab51f20b53989c7bcbc923f956b5a16556bc6a4c143265e280769f12792d0e0913e",
			seedKorean:             "62392a9144379952afbcdd70c7e68f1a8ab06cc6fec4f0fe22915b8b26b0939061f31ae0c761579681bc0b3619fca8c8a27dcd9f964ab694068cac04f26de6ac",
			seedSpanish:            "338e1ee586e109e80a53af2294bca03f4a5a7e9d089f04d1f02b30dde370c8ae4268a37909bd278c21e29fc24e2a3f30104eb8dd153192eda5646415dbc21fc0",
		},
		{
			entropy:                "b63a9c59a6e641f288ebc103017f1da9f8290b3da6bdef7b",
			mnemEnglish:            "renew stay biology evidence goat welcome casual join adapt armor shuffle fault little machine walk stumble urge swap",
			mnemChineseSimplified:  "芽 碗 想 富 训 粪 争 额 生 使 怒 阿 折 泥 剑 勾 傅 浇",
			mnemChineseTraditional: "芽 碗 想 富 訓 糞 爭 額 生 使 怒 阿 折 泥 劍 勾 傅 澆",
			mnemItalian:            "rimbalzo solubile avvenire fanfara idra vicenda calibro malto adipe anatra scuderia focaccia monetario mummia velcro spatola uditivo staffa",
			mnemJapanese:           "ぬすむ ふっかつ うどん こうりつ しつじ りょうり おたがい せもたれ あつめる いちりゅう はんしゃ ごますり そんけい たいちょう らしんばん ぶんせき やすみ ほいく",
			mnemKorean:             "재정 체온 교통 번역 새벽 홀로 꽃잎 시금치 간접 경제 중반 본사 아시아 알코올 현상 최선 학위 치약",
			mnemSpanish:            "pleito semana ático ensayo giro viaje buceo júpiter activo amigo repetir fábula llover madera veinte siete trompa soplar",
			seedEnglish:            "9248d83e06f4cd98debf5b6f010542760df925ce46cf38a1bdb4e4de7d21f5c39366941c69e1bdbf2966e0f6e6dbece898a0e2f0a4c2b3e640953dfe8b7bbdc5",
			seedChineseSimplified:  "4e62ea1e33462a4b756e1a1c9fdd921906e3a92e7a6d8b3aadef46ab0a6a1401af4ab6ee76588567505d110b8baa9098a162613c1329efdc6fa119ba61d413d0",
			seedChineseTraditional: "73f34390a71ce9d84c2bcd5137fc39520a1ddaa77db53601211fea7e217a971be45fe41d52ff94f8974ffc1179056d7d6b36916f4f9820acc58f3dec97b65732",
			seedItalian:            "f988c804b5adc0dda6bfc42343cc22f1a3bb53fa41a7b0cae7f059d759549f2b2911caa32c66a1a04b2bccc50cf669336af82491741a816b8595aa9cc97dbadc",
			seedJapanese:           "a5fe510d0485f7d74dec53fbc1aeb7bf3d527075dcc5ef657e0b3a8ff613554228099faa1cc9332f9a1dde264cefa6493f70ca3828c514781e78dd7c5e39877d",
			seedKorean:             "84a175cbea67eeb84bde6fc217eaa323059b1514be1fa2981dfee7faf0f2de8d5158a9e12c3e562a1d27eb740ccecdd128ddec83483e4690018a3b9d95632a5c",
			seedSpanish:            "12e9454bfe0cb26cb91db194f7be1297ea0f0ff07038f9f70fc3364a85f4196991b01c7ec84ebc91f0611597c8b346cd20e2623ce8c0af8e4040cf7bc05f2218",
		},
		{
			entropy:                "3e141609b97933b66a060dcddc71fad1d91677db872031e85f4c015c5e7e8982",
			mnemEnglish:            "dignity pass list indicate nasty swamp pool script soccer toe leaf photo multiply desk host tomato cradle drill spread actor shine dismiss champion exotic",
			mnemChineseSimplified:  "严 勒 伸 销 男 佛 锋 忍 啥 弓 横 泡 综 圆 概 坑 断 台 鸟 来 簧 尔 美 初",
			mnemChineseTraditional: "嚴 勒 伸 銷 男 佛 鋒 忍 啥 弓 橫 泡 綜 圓 概 坑 斷 台 鳥 來 簧 爾 美 初",
			mnemItalian:            "disumano pigro mondina lingua ornativo stacco prenotare saziato sfratto tavolata microbo podismo operato digitale lacca telefono coricato educare snellire addome sclerare dolce cappero feltro",
			mnemJapanese:           "くのう てぬぐい そんかい すろっと ちきゅう ほあん とさか はくしゅ ひびく みえる そざい てんすう たんぴん くしょう すいようび みけん きさらぎ げざん ふくざつ あつかう はやい くろう おやゆび こすう",
			mnemKorean:             "목사 위협 아스팔트 수준 영향 취향 이전 조명 질서 통제 실력 의견 열심히 명의 소풍 퇴근 대합실 물질 천둥 간부 주전자 몸짓 낭비 변신",
			mnemSpanish:            "cúpula odiar llorar inicio moreno sopa ozono rápido rotar tejer libro opción moho cubrir horno tema cigarro diadema sardina acné relato dátil cacao espejo",
			seedEnglish:            "ff7f3184df8696d8bef94b6c03114dbee0ef89ff938712301d27ed8336ca89ef9635da20af07d4175f2bf5f3de130f39c9d9e8dd0472489c19b1a020a940da67",
			seedChineseSimplified:  "1e6a232b629f0708abbc19d92d7bda1f9ec659003c42769f62f38d1336bea5f0a3ed77475f8c0e75170980b12b7a782aec799ba8c24821f5872ac60a94177f50",
			seedChineseTraditional: "f4728e7f4c8664bf908dd073a8ad025b492cf65a15500d471497d8644daf08cf7179a91523654a2a0c0872065b89d33b1cbe811a731ca365ee8a4c2405e34a58",
			seedItalian:            "41d464af9fb1f2222011ac4fa96777be87ac121b28e3dd3aaedfa243a68b2b8c3e131c5643c344e0c967adc39145683480da53a33ff138383cddd67a68d061f7",
			seedJapanese:           "3ca539f28db49e01d56b8dca1b513131dcd57833e961caabad88b7bbf2347ce5ece844c025bc88bd7a90fe4069a5ce2115f5571da9021af64e782539267fc687",
			seedKorean:             "ed4535b5e5f0d8bebc65c817fc9791787f21ef9f2870f25e3e21bc7643fcfbf76a540508d910fe82c4d7666abcf4d90e6dd1fccbb8f2713ae7c4abb60f05e3bb",
			seedSpanish:            "acb2b4e604937ce8bbd1048577fc9cc4f864551d28772f572068b6749ddbd38a9afcb189a62453ceae15542cc1af7e9e5372e62d113a6db88d5250ab6afce4f1",
		},
		{
			entropy:                "0460ef47585604c5660618db2e6a7e7f",
			mnemEnglish:            "afford alter spike radar gate glance object seek swamp infant panel yellow",
			mnemChineseSimplified:  "可 所 筹 铝 货 纸 嘴 乳 佛 居 旅 卿",
			mnemChineseTraditional: "可 所 籌 鋁 貨 紙 嘴 乳 佛 居 旅 卿",
			mnemItalian:            "agente allegro slogatura reddito gommone guadagno palesare sbrinare stacco lirica pianta zircone",
			mnemJapanese:           "あみもの いきおい ふいうち にげる ざんしょ じかん ついか はたん ほあん すんぽう てちがい わかめ",
			mnemKorean:             "감정 거실 채널 자정 사흘 상식 온갖 졸음 취향 수컷 월드컵 흔히",
			mnemSpanish:            "aduana ajuste samba perder gafas gen natal rebote sopa innato ochenta zafiro",
			seedEnglish:            "65f93a9f36b6c85cbe634ffc1f99f2b82cbb10b31edc7f087b4f6cb9e976e9faf76ff41f8f27c99afdf38f7a303ba1136ee48a4c1e7fcd3dba7aa876113a36e4",
			seedChineseSimplified:  "0ecc4917f75f06bf73bddb4064fab59a3ed15af37b0d0e6fb89f27b974b8d0311a60c9b2c09115eb2f4ba8c49a3fcf7b792b7f20a5de2ad22c2597c23abc29e8",
			seedChineseTraditional: "1ffaf0e925cf9a8fd7e9392324a7e3e25bb77c0af38ba8782ce878275b452694cac9993f758b673233a9fca1d336ab5a39ff29ec53bb526bed7b8dd30c2b94c1",
			seedItalian:            "a11334b5645da8c9eaa166429c1bfee321f80eaf02b7e055224fdb65f0f2fa72d07be9237130ee5e1bda51be02305afa9460e6c030c8495b5985d84dbda59dda",
			seedJapanese:           "1bd33e347a219ff2ff2dbacc0c6149a97d09e20f7dd4951552e1516eb865710387dc011c22b256270661094ff9bfb080b939eb6dd1cb8705afabe0f38cf3b74d",
			seedKorean:             "fd9f965f624b20b10b4c5e38cd237bfce5a1be914032ce084c5072357a755055107ede64918ba2a3a5845484513f3e5c8e3d5ee89edaed5668b350a8f13ce5f7",
			seedSpanish:            "fbeec9484d0ba972601190f2201049c522c1b24b8a3584478f2ca11dd58683c232241df21dca593f0beb1c9842323f81c9fd53d19d9af1be7686424c746711b6",
		},
		{
			entropy:                "72f60ebac5dd8add8d2a25a797102c3ce21bc029c200076f",
			mnemEnglish:            "indicate race push merry suffer human cruise dwarf pole review arch keep canvas theme poem divorce alter left",
			mnemChineseSimplified:  "销 仿 喊 忽 姆 皇 感 供 授 隆 量 岩 造 岗 泵 推 所 堂",
			mnemChineseTraditional: "銷 仿 喊 忽 姆 皇 感 供 授 隆 量 岩 造 崗 泵 推 所 堂",
			mnemItalian:            "lingua recondito rapato nucleo spessore lampo croce elsa prefisso rischio ampio maratona bubbone svagare prassi dormire allegro milano",
			mnemJapanese:           "すろっと にくしみ なやむ たとえる へいこう すくう きない けってい とくべつ ねっしん いたみ せんせい おくりがな まかい とくい けあな いきおい そそぐ",
			mnemKorean:             "수준 자율 입시 에너지 추측 손길 동화책 민주 이웃 적응 경력 시인 기준 클럽 이성 무덤 거실 실습",
			mnemSpanish:            "inicio pera pelar medio simio hueso cocina directo óvulo pompa amante lágrima bóveda talento ostra defensa ajuste lienzo",
			seedEnglish:            "3bbf9daa0dfad8229786ace5ddb4e00fa98a044ae4c4975ffd5e094dba9e0bb289349dbe2091761f30f382d4e35c4a670ee8ab50758d2c55881be69e327117ba",
			seedChineseSimplified:  "402b0348f2c1cfb2bed9f1b35038b3858fdef84fcf1b5145aee02bd95f2fa5d8a8fe5591100fa3e13df296de9479b78cd2a256d674b7659c52658c25b10901ac",
			seedChineseTraditional: "049a53d601580da9c0050a2c2972bdc12ba3e5c73642f84c415cdb9f4f4b077fac754567e286adfc55d4fe99ba861eddc4837d5365c62a18e580c1d0167a4708",
			seedItalian:            "5b6891b038e178a92117b8ac854e6cfd2d482916fd2f2990eadc6de885614e1b8ffd118586afc7ffea78e680399acfafa9f8db8430be7160cebc80451629c077",
			seedJapanese:           "37a76adf17a8330e495ea6e8b41cbb590ae7672a48bbcae709483b4a0b1b5104cacc5c5df6595a9de22c0116a33138233d15ede90c4fc7ba7cb97488d168c137",
			seedKorean:             "bdaf23a011e1ac722308c543ac64e2f126a52f685975044185e972965c674d8e96dffb30dca5448c1e27f3742bfb54700f70c809eda5c6fd8a31f242b19d47ab",
			seedSpanish:            "26ec835839a0556796cb2f483ea6965cfa845a059867df950a8314d0d7edca4eacb1076e4aa7977d321ae90da1a29893c2025e2f585d4839637fefed3abc1f26",
		},
		{
			entropy:                "2c85efc7f24ee4573d2b81a6ec66cee209b2dcbd09d8eddc51e0215b0b68e416",
			mnemEnglish:            "clutch control vehicle tonight unusual clog visa ice plunge glimpse recipe series open hour vintage deposit universe tip job dress radar refuse motion taste",
			mnemChineseSimplified:  "况 越 慌 叙 斑 信 缆 扬 忘 吗 抱 舰 抵 怕 闷 状 宴 煮 胡 告 铝 寄 尘 孤",
			mnemChineseTraditional: "況 越 慌 敘 斑 信 纜 揚 忘 嗎 抱 艦 抵 怕 悶 狀 宴 煮 胡 告 鋁 寄 塵 孤",
			mnemItalian:            "circa commando urgenza tendone tunisia chirurgo vangare lavoro pranzo gufo ribelle scapola peccato lacrima valoroso devoto tubatura tardivo malsano edile reddito ricordo ombra stufo",
			mnemJapanese:           "かほご きうい ゆたか みすえる もらう がっこう よそう ずっと ときどき したうけ にんか はっこう つみき すうじつ よけい くげん もくてき まわり せめる げざい にげる にんたい たんそく ほそく",
			mnemKorean:             "단위 대단히 할인 트럭 학력 다이어트 햇살 솜씨 이상 상점 장례 좌석 왼손 속담 핵심 며느리 학교 토요일 시골 물리학 자정 장비 연장 콘서트",
			mnemSpanish:            "castor cetro úlcera tender tren carne vaina icono oso geranio piloto red nivel hoyo vacío croqueta trazar tauro juntar día perder piojo miseria sur",
			seedEnglish:            "fe908f96f46668b2d5b37d82f558c77ed0d69dd0e7e043a5b0511c48c2f1064694a956f86360c93dd04052a8899497ce9e985ebe0c8c52b955e6ae86d4ff4449",
			seedChineseSimplified:  "bd5c11fbf4dadb6098691ad9aa111879fb6ac5452aa56988d1623f08b5533be6d3cd1f192cb78574168f885e514d702e626b465bc011e7539c75fa36914ddc92",
			seedChineseTraditional: "245c0079ed3f521170d2680b0195459eb69cd1e11715b657eeca71480d234c0e8ba412f4b2de0388e9a16e7df8dbbfcd17634a9fe362232369f01b81ee0804f7",
			seedItalian:            "bdceb85bbe1da2c2fe44dff7ff67aa58899c2c78dce4521e9d23bcb65231345ee25bb3ab5182b6c4325d0d9a946cb96a7c1649e27f8d1ab8e824aaa825d8e8c9",
			seedJapanese:           "ba369b6718743db50a501ca4bc452763b9230370e923063cd7be7fafaf537c7fadd677cfd2066f78c752f5d5830fb3794983b7e896d58722d559e26060b44309",
			seedKorean:             "3f387663035d904317f4dea874874db2c56614d71a566a9af698738b0f822a745e02afdb567980f2154b64ab5a0ff9cd94007354b3da5f4c43801254c93f5c95",
			seedSpanish:            "e030c576214c756d847e79429be634d2054cb489f37f01d892a7393cc368927bd6af4203c96aa34e237fcb96365b7d4ed02e20c518818a12944efde5fc6e6ea4",
		},
		{
			entropy:                "eaebabb2383351fd31d703840b32e9e2",
			mnemEnglish:            "turtle front uncle idea crush write shrug there lottery flower risk shell",
			mnemChineseSimplified:  "惩 若 呵 希 团 曰 隙 盗 塔 友 牵 牌",
			mnemChineseTraditional: "懲 若 呵 希 團 曰 隙 盜 塔 友 牽 牌",
			mnemItalian:            "trapano genotipo trio leggero cruciale zenzero scrutinio svelare motto furgone rivincita scindere",
			mnemJapanese:           "めいえん さのう めだつ すてる きぬごし ろんぱ はんこ まける たいおう さかいし ねんいり はぶらし",
			mnemKorean:             "플라스틱 사계절 하룻밤 송이 딸아이 흐름 중독 타자기 악몽 불안 전주 주식",
			mnemSpanish:            "tórax fracaso trabajo idioma codo yeso reparto tamaño lucha fila prensa rehén",
			seedEnglish:            "bdfb76a0759f301b0b899a1e3985227e53b3f51e67e3f2a65363caedf3e32fde42a66c404f18d7b05818c95ef3ca1e5146646856c461c073169467511680876c",
			seedChineseSimplified:  "41516e14e79ebe65e726c50e3aa42ec9d5ecf621a526ad49eb7dc18d8b85058f27a620d6ee9e3037f7ad936651a43f73659158d09c108c926419161932d9f1d3",
			seedChineseTraditional: "15d6cbca0bcd6e687ea7c68f3a573418bd94e4e1d4221d2bce7185af7f913b71146312aeecb599fc981813c46d4abecf86d2cc1e607d423ec5822300effb7625",
			seedItalian:            "9357d82a70821589215d4a150d9a75e9be4c765cd9eeb530a78911bd42e647eed1a5b3f6a88344e94067c92dd788293b07827e69f88e03b03c14572c1c6c4d14",
			seedJapanese:           "065cfeac3b160a68307b6a4d5879b6c8f7ed6c9de396abb8bbd26f4dde61c4b45f5977187bd69a228cd521fd0d901a80df90df07a8115c3de05831e549b14b4a",
			seedKorean:             "0358feefe6fd5dac8688aaf52090b1e1696c83e2844f640341c02f74d7183849b3b9300b86e95aecaaf197c046da8e95012cfa8cae1ee992cf4a8e8210af798a",
			seedSpanish:            "a5083e544700dc9933be40a727afdd373a4e417b4ec97b1382c2758836320a8b3d16d06a4d649d8173544867bb59cd89528024a14aac0a40dc6026502bd96020",
		},
		{
			entropy:                "7ac45cfe7722ee6c7ba84fbc2d5bd61b45cb2fe5eb65aa78",
			mnemEnglish:            "kiss carry display unusual confirm curtain upgrade antique rotate hello void custom frequent obey nut hole price segment",
			mnemChineseSimplified:  "探 器 讲 斑 叫 构 醇 自 矩 弦 柄 太 央 筒 婚 松 怪 邓",
			mnemChineseTraditional: "探 器 講 斑 叫 構 醇 自 矩 弦 柄 太 央 筒 婚 松 怪 鄧",
			mnemItalian:            "materasso busta domenica tunisia coltivato curvo tuta ameba rompere intasato varcato dado gemello palazzina paga irrigato prova sbruffone",
			mnemJapanese:           "せんぱい おしえる ぐんかん もらう きあい きぼう やおや いせえび のいず じゅしん よゆう きみつ さといも ちんもく ちわわ しんせいじ とめる はちみつ",
			mnemKorean:             "시집 깍두기 몹시 학력 당연히 마요네즈 학비 결론 점원 세금 향상 마이크 빛깔 옥수수 오히려 소망 인종 종교",
			mnemSpanish:            "langosta broma débil tren cero colgar tribu almíbar prole hebra vampiro colmo forro nasal nariz historia pañuelo recaer",
			seedEnglish:            "ed56ff6c833c07982eb7119a8f48fd363c4a9b1601cd2de736b01045c5eb8ab4f57b079403485d1c4924f0790dc10a971763337cb9f9c62226f64fff26397c79",
			seedChineseSimplified:  "47fda4426598bc3c9b274d01c314c99cd391652813475d0005699c1c93f0205e50b4c38a96c436fd60a4aa58ee14f88e627569c4341fc9f30c496da2e7465cf1",
			seedChineseTraditional: "cc7e9efb7ec3e190ee600e574b0434a268c4bd229c81e8adae1e0a89f8ed957fe270b841309e77faeffa2562bd305b171a7b1e7ae6a272b0cf6eced201db8bac",
			seedItalian:            "67f58f2f0ecf0fb099d7edaa0c289b374d95a2ea100de1637af11a3b30bcb5639a8b5527235bc4400466333c687924593b87dfc2f15dd60d22cdc972395511c7",
			seedJapanese:           "a3e06b761cd1ddde4f652856c495b53c67f84e23a545f0a97b79f94e84ebcab5999439124275e2e118cb03d34772f5b03bb2d3d048a532e019aa6e7121b39b9c",
			seedKorean:             "6938637bd9580bf4aa776502e21ed4563f1a627127feb4ec18b08eb25eeebd55a4b641b3f96b425938892544cd62455a36e95c8df2c1fde82bcca6545b41b694",
			seedSpanish:            "be98fe494599826bd0056d02596eccee914ead5b8bd6387920663e813d3965ae1d9f0ca0c2eba3f888a2ddd41736cb2dc25ea5ee625e09b69e067edc2a0729fb",
		},
		{
			entropy:                "4fa1a8bc3e6d80ee1316050e862c1812031493212b7ec3f3bb1b08f168cabeef",
			mnemEnglish:            "exile ask congress lamp submit jacket era scheme attend cousin alcohol catch course end lucky hurt sentence oven short ball bird grab wing top",
			mnemChineseSimplified:  "升 它 且 归 蒋 剧 修 伐 天 商 产 油 际 护 旋 尼 乌 墙 洛 明 已 脱 酱 罐",
			mnemChineseTraditional: "昇 它 且 歸 蔣 劇 修 伐 天 商 產 油 際 護 旋 尼 烏 牆 洛 明 已 脫 醬 罐",
			mnemItalian:            "fede annegare colza mensola specie magico europa sarto apparire coppia albo cambusa copione esercito mucosa latino scandalo perno scossone arso avviso imballo vissuto tentacolo",
			mnemJapanese:           "こころ いどう きあつ そうがんきょう へいあん せつりつ ごうせい はいち いびき きこく あんい おちつく きこえる けんとう たいこ すすめる はっけん ていど はんおん いんさつ うなぎ しねま れいぼう みつかる",
			mnemKorean:             "변경 계약 당장 신고 최종 습기 배달 제주도 고민 대충 강제 나머지 대출 발톱 안내 손톱 종합 울산 중계방송 공짜 교환 생일 환자 특성",
			mnemSpanish:            "esfera ángulo cerrar leer sílaba juez encargo ración anuncio cielo agrio buey ciego educar lunes hundir recurso número remo área atleta gorila visor tenso",
			seedEnglish:            "095ee6f817b4c2cb30a5a797360a81a40ab0f9a4e25ecd672a3f58a0b5ba0687c096a6b14d2c0deb3bdefce4f61d01ae07417d502429352e27695163f7447a8c",
			seedChineseSimplified:  "137a41c649798f8dcb9a46378bf74c67ebfffbd8fcea04b34721fa5bc89eed726c46a1af50825dfb14196362814568a5be8bb418680b64a6213309e2bc6d5bc3",
			seedChineseTraditional: "7b18d49c2bcc8cbbd8ff869162a0c3ca7a0f0855ef6e8a29fa55ff8181827657ff6b8b30bae395aaa5073adcebde22dc5e65dfaadd9431bfd32088c59882c46c",
			seedItalian:            "759e5b5b4b2810c8314ed23166e733cd879f4d81c3ddd0e02ae54bb1eae3938b9637fffc02f3a20064a2a9ccb8581e576c4f9e6d41f301d9cddfbbcb727de717",
			seedJapanese:           "37ed8facbb2fcad238893671e9e12fe25f612f1ec5c39c38f3c0b332d6e5b9fb38902dfc9b3e664029a13adab9e8a1ed5869ed9d0a5854974dd5f608676064b7",
			seedKorean:             "6fd7ad6ed0712293a9d3c3bd8d78941db619e3541e0ae8f5dc7d9d192b9c72e55a197bad0c05abc99db58144e5a614e31c1dde2086baabb2e16c17d5ddc150c8",
			seedSpanish:            "337858f949a2f0fe56c0d9995c768af0237036751e2b7b09e9c60a6f5263e2499319f5702b3bdeb19e7a424f2ebe42d2f3746faf26520ae7a2173d623b4a2581",
		},
		{
			entropy:                "18ab19a9f54a9274f03e5209a2ac8a91",
			mnemEnglish:            "board flee heavy tunnel powder denial science ski answer betray cargo cat",
			mnemChineseSimplified:  "常 诉 握 仗 窗 层 疗 赏 化 系 东 济",
			mnemChineseTraditional: "常 訴 握 仗 窗 層 療 賞 化 系 東 濟",
			mnemItalian:            "ballata fumetto insieme tralcio procura descritto satellite senso ambito attuale burrasca calmo",
			mnemJapanese:           "うりきれ さいせい じゆう むろん とどける ぐうたら はいれつ ひけつ いずれ うちあわせ おさめる おたく",
			mnemKorean:             "국립 불과 성적 풍습 인근 먹이 제품 지름길 결과 교과서 김치 나들이",
			mnemSpanish:            "avena fiel haz topar palco crimen raíz rigor alma astuto brisa bucle",
			seedEnglish:            "6eff1bb21562918509c73cb990260db07c0ce34ff0e3cc4a8cb3276129fbcb300bddfe005831350efd633909f476c45c88253276d9fd0df6ef48609e8bb7dca8",
			seedChineseSimplified:  "b14c71e5c6fececc7ee482bacbf4e5b3f1861c425378db96fd893e7002ac7a01108e8933a03a317f7f0bc1a48474e21291c899b149c35b3dc9555401be7858ef",
			seedChineseTraditional: "03477bcacf4e289bbdd0fc8924cc8491dd5011df3b91c5b4a7cfb3fc44944422ed0294a05a889252351ff41095a3fcc1c5696b10bf33ff02cc769e8a4a99c661",
			seedItalian:            "90fb045633be02430f26492f543c91fcef606a5c80d85774897244cf9ca10a6148a76af2f8562b555326d0c91e299f273d53b1e34953774854b343023c562aba",
			seedJapanese:           "db0b8914d12023ea9c2ffacca9e98cde2afd22aa636811c1043ec5df842c8f8f71a5425b7c2d579d88e214f5c27f4a24b940666c6c8542b5b46414ad8e023930",
			seedKorean:             "3f91644673d1ce366b5e83378ddab52ea73922a4eee0acb6d559ff8f24093aa4280f4e7a1eaa4ab166304ed2a3a3b281a3ae0e872a15f94cc540300bf514d090",
			seedSpanish:            "805b75dfa5021feb4212af6508364acb71bc26f3ae3e1b04d46997da276ffb3698b55986d20eaf26d60d8ab4a57fbebb6caed0d63cd68e5f2ce523880e5082df",
		},
		{
			entropy:                "18a2e1d81b8ecfb2a333adcb0c17a5b9eb76cc5d05db91a4",
			mnemEnglish:            "board blade invite damage undo sun mimic interest slam gaze truly inherit resist great inject rocket museum chief",
			mnemChineseSimplified:  "常 直 顾 号 雅 雕 粗 乡 浙 阻 脆 呼 虎 渐 景 诚 吴 安",
			mnemChineseTraditional: "常 直 顧 號 雅 雕 粗 鄉 浙 阻 脆 呼 虎 漸 景 誠 吳 安",
			mnemItalian:            "ballata azzimo lusinga daniela trivella spillato obbligo lungo sereno governo tortora livrea rinuncia impacco lode rodaggio opposto cassone",
			mnemJapanese:           "うりきれ うねる せっさたくま きもち めんきょ へいたく たまご ぜっく びじゅつかん さんそ むせる せいじ ねくたい しはらい せおう ねんど たんまつ がいけん",
			mnemKorean:             "국립 구멍 스위치 마찰 하순 출근 여덟 스스로 지우개 산업 포함 수화기 저렇게 서양 숙소 절반 열차 노동",
			mnemSpanish:            "avena atún jeringa comida tráfico sobre mente jaula ritmo gala tobillo íntimo poesía grano inútil probar molde calle",
			seedEnglish:            "f84521c777a13b61564234bf8f8b62b3afce27fc4062b51bb5e62bdfecb23864ee6ecf07c1d5a97c0834307c5c852d8ceb88e7c97923c0a3b496bedd4e5f88a9",
			seedChineseSimplified:  "ba4fc6c54ff8e226b9932394b8278d0a8cca13361a4e2feb33a2d77ece70915c26b430b4736d87db4f52c10a8abc0ad3bf9b93daf058fbbb44346acb765eb745",
			seedChineseTraditional: "d63c03f4b9d417421724e458a93e486981f514e9114013cc7259711c47150d7977fa2afdf2e965d3b4540a594e0f001fd9fa7bcf70b674305fb7ef4762a8a077",
			seedItalian:            "b317b7e1cd3bfe131bacf41eb596e6b68ec368484692163ed24c1c8db75391e3eeec4bc9f6acc540e30aa0c09015d320c0eba571951804945b9944c773e81d3d",
			seedJapanese:           "6a6436f5a2353a9fc8f091d49bedc6f51ca23987dc32ea9798786a2d94191146f36604aecffd8494db8c5eac7e858e7e17e1e2eeae8b7dead483e02ea9c939a6",
			seedKorean:             "1460fd60cf80eeb543d336d7ca1e272ddb9ccb78a5815274bc9074f7a0c3c858756144df9d2daacc60ea1c79dbb17d4eebea9af3afc2fd03c9a89444e55e89a8",
			seedSpanish:            "82509727ea09696854191b68976f202411fcf6cfa26187bbf5bf3fe966f12fe2d13629ed71eafed0624db2a5b2214b80b3394c910d87801b7f6844b29c9e901d",
		},
		{
			entropy:                "15da872c95a13dd738fbf50e427583ad61f18fd99f628c417a61cf8343c90419",
			mnemEnglish:            "beyond stage sleep clip because twist token leaf atom beauty genius food business side grid unable middle armed observe pair crouch tonight away coconut",
			mnemChineseSimplified:  "情 韩 貌 科 此 飘 杰 横 前 命 普 混 干 肩 欢 烷 愈 当 朗 柱 约 叙 与 温",
			mnemChineseTraditional: "情 韓 貌 科 此 飄 傑 橫 前 命 普 混 幹 肩 歡 烷 愈 當 朗 柱 約 敘 與 溫",
			mnemItalian:            "autista sogno serio chimera assurdo treccia tecnico microbo apertura assoluto grado gamma bordo scusare impiego trillo nuvola anarchia palude pettine criceto tendone ardito cittadino",
			mnemJapanese:           "うちゅう ふそく ひしょ がちょう うけもつ めいそう みかん そざい いばる うけとる さんま さこつ おうさま ぱんつ しひょう めした たはつ いちぶ つうじょう てさぎょう きつね みすえる いりぐち かめれおん",
			mnemKorean:             "교실 청년 지원 다양성 관람 필수 통화 실력 고등학생 관념 살림 비만 긍정적 중순 서적 하늘 여관 경쟁 온종일 원인 독립 트럭 공군 단추",
			mnemSpanish:            "atajo secta rito carga asalto torpedo teléfono libro anual asado gallo flauta boa rescate gratis toser melón ameno náusea obvio clínica tender apuro caudal",
			seedEnglish:            "b15509eaa2d09d3efd3e006ef42151b30367dc6e3aa5e44caba3fe4d3e352e65101fbdb86a96776b91946ff06f8eac594dc6ee1d3e82a42dfe1b40fef6bcc3fd",
			seedChineseSimplified:  "01204593c1558eb4701c18c476c5fa27cd8076bd218a11d848a87417a7012b02404320b132f891c8ea9108a366a6ab383ce2958d9a426d1474a1fbdade6e9ce9",
			seedChineseTraditional: "94fcad39535a29ef0b6024ff78c18933f721c285651d52d13e026ad91ae7608491d579da0c7dace3ea5b17aeb16d9c9e1ad8b9647c9bf3968441d775c15aaf51",
			seedItalian:            "457df84d1553fded17969444f8cee1ccce9cf3306cd23d79f8c0c9025960688abca3e413eded27776de38208393efda567078809d5f67569a10e5ff0d9d7d6c2",
			seedJapanese:           "37ff351d26601c20cab59aed72ba7cdff4bd485fdb70fc2bb25c96d6815ce6c506468cc3fc4bd233cd67affa04bd759c29d61ac3e18db0a4301ef28ef230e792",
			seedKorean:             "59d50acbde7a5802b9c9136a24529cb7b65906656c1868c17a95e7fcd1ca6d8d84ed6e87d77eb6c4226e9313e36e53766b3a995408431bb87c77aeacea8a5606",
			seedSpanish:            "9f99ae125b87b67703d85562f90a95c2f72066a3bc39e7b4578c7f79856949f3fd4acf976743b9be9cac0e2e1063e7bc86ca8ddffcc2b67efcc8b31d69adc067",
		},
	}
}

func TestEntropyFromMnemonic_128(t *testing.T) {
	testEntropyFromMnemonic(t, 128)
}

func TestEntropyFromMnemonic_160(t *testing.T) {
	testEntropyFromMnemonic(t, 160)
}

func TestEntropyFromMnemonic_192(t *testing.T) {
	testEntropyFromMnemonic(t, 192)
}

func TestEntropyFromMnemonic_224(t *testing.T) {
	testEntropyFromMnemonic(t, 224)
}

func TestEntropyFromMnemonic_256(t *testing.T) {
	testEntropyFromMnemonic(t, 256)
}

func testEntropyFromMnemonic(t *testing.T, bitSize int) {
	for i := 0; i < 512; i++ {
		entropy, err := NewEntropy(bitSize)
		assertNil(t, err)
		assertTrue(t, len(entropy) != 0)

		mnemonic, err := NewMnemonic(entropy, "en")
		assertNil(t, err)
		assertTrue(t, len(mnemonic) != 0)

		outEntropy, err := EntropyFromMnemonic(mnemonic, "en")
		assertNil(t, err)
		assertEqualByteSlices(t, entropy, outEntropy)
	}
}
