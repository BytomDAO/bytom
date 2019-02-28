package pseudohsm

import (
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/bytom/crypto/ed25519/chainkd"
)

func TestRestore(t *testing.T) {
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	hsm, err := New(dirPath)
	if err != nil {
		t.Fatal("create hsm err:", err)
	}

	testCases := []struct {
		image    string
		wantErr  error
		wantKeys []string
	}{
		{
			image:    "{\"xkeys\":[{\"alias\":\"test4\",\"crypto\":{\"cipher\":\"aes-128-ctr\",\"cipherparams\":{\"iv\":\"45650919af7c61d907d8681ae8a3f8d0\"},\"ciphertext\":\"0d073355814763f3c4d0c049668d362a419af56ff193a1e65f1e867babaf3b8c9e91e539918314e7b78532216093f8bc2dd7712af579f7ff2b95ef0ecb8562a0\",\"kdf\":\"scrypt\",\"kdfparams\":{\"dklen\":32,\"n\":4096,\"p\":6,\"r\":8,\"salt\":\"d6a5ff1f5ba2d678db428cb2d387a8130f0223c9fe271dfdf02a3edca4608b30\"},\"mac\":\"889920bee3829a8b3aaccb8c6e8aa5f9dda7f986d72efa25c5777c88c1a86c17\"},\"id\":\"868047f4-6613-4abb-902a-4b2a288cd8c7\",\"type\":\"bytom_kd\",\"version\":1,\"xpub\":\"a57f2ac07c69a71ec2ec7432c573c8b0680f3b6e4bb3c30baaf845b2685840b133dbe260c332640904f0a523421cf911084e98dd18204e88271fbc1c35f18fb6\"}]}",
			wantErr:  nil,
			wantKeys: []string{"a57f2ac07c69a71ec2ec7432c573c8b0680f3b6e4bb3c30baaf845b2685840b133dbe260c332640904f0a523421cf911084e98dd18204e88271fbc1c35f18fb6"},
		},
		{
			image:    "{\"xkeys\":[{\"alias\":\"test4\",\"crypto\":{\"cipher\":\"aes-128-ctr\",\"cipherparams\":{\"iv\":\"45650919af7c61d907d8681ae8a3f8d0\"},\"ciphertext\":\"0d073355814763f3c4d0c049668d362a419af56ff193a1e65f1e867babaf3b8c9e91e539918314e7b78532216093f8bc2dd7712af579f7ff2b95ef0ecb8562a0\",\"kdf\":\"scrypt\",\"kdfparams\":{\"dklen\":32,\"n\":4096,\"p\":6,\"r\":8,\"salt\":\"d6a5ff1f5ba2d678db428cb2d387a8130f0223c9fe271dfdf02a3edca4608b30\"},\"mac\":\"889920bee3829a8b3aaccb8c6e8aa5f9dda7f986d72efa25c5777c88c1a86c17\"},\"id\":\"868047f4-6613-4abb-902a-4b2a288cd8c7\",\"type\":\"bytom_kd\",\"version\":1,\"xpub\":\"a57f2ac07c69a71ec2ec7432c573c8b0680f3b6e4bb3c30baaf845b2685840b133dbe260c332640904f0a523421cf911084e98dd18204e88271fbc1c35f18fb6\"},{\"alias\":\"test4\",\"crypto\":{\"cipher\":\"aes-128-ctr\",\"cipherparams\":{\"iv\":\"45650919af7c61d907d8681ae8a3f8d0\"},\"ciphertext\":\"0d073355814763f3c4d0c049668d362a419af56ff193a1e65f1e867babaf3b8c9e91e539918314e7b78532216093f8bc2dd7712af579f7ff2b95ef0ecb8562a0\",\"kdf\":\"scrypt\",\"kdfparams\":{\"dklen\":32,\"n\":4096,\"p\":6,\"r\":8,\"salt\":\"d6a5ff1f5ba2d678db428cb2d387a8130f0223c9fe271dfdf02a3edca4608b30\"},\"mac\":\"889920bee3829a8b3aaccb8c6e8aa5f9dda7f986d72efa25c5777c88c1a86c17\"},\"id\":\"868047f4-6613-4abb-902a-4b2a288cd8c7\",\"type\":\"bytom_kd\",\"version\":1,\"xpub\":\"a57f2ac07c69a71ec2ec7432c573c8b0680f3b6e4bb3c30baaf845b2685840b133dbe260c332640904f0a523421cf911084e98dd18204e88271fbc1c35f18fb6\"}]}",
			wantErr:  nil,
			wantKeys: []string{"a57f2ac07c69a71ec2ec7432c573c8b0680f3b6e4bb3c30baaf845b2685840b133dbe260c332640904f0a523421cf911084e98dd18204e88271fbc1c35f18fb6"},
		},
		{
			image:    "{\"xkeys\":[{\"alias\":\"test4\",\"crypto\":{\"cipher\":\"aes-128-ctr\",\"cipherparams\":{\"iv\":\"45650919af7c61d907d8681ae8a3f8d0\"},\"ciphertext\":\"0d073355814763f3c4d0c049668d362a419af56ff193a1e65f1e867babaf3b8c9e91e539918314e7b78532216093f8bc2dd7712af579f7ff2b95ef0ecb8562a0\",\"kdf\":\"scrypt\",\"kdfparams\":{\"dklen\":32,\"n\":4096,\"p\":6,\"r\":8,\"salt\":\"d6a5ff1f5ba2d678db428cb2d387a8130f0223c9fe271dfdf02a3edca4608b30\"},\"mac\":\"889920bee3829a8b3aaccb8c6e8aa5f9dda7f986d72efa25c5777c88c1a86c17\"},\"id\":\"868047f4-6613-4abb-902a-4b2a288cd8c7\",\"type\":\"bytom_kd\",\"version\":1,\"xpub\":\"a57f2ac07c69a71ec2ec7432c573c8b0680f3b6e4bb3c30baaf845b2685840b133dbe260c332640904f0a523421cf911084e98dd18204e88271fbc1c35f18fb\"}]}",
			wantErr:  ErrXPubFormat,
			wantKeys: []string{"a57f2ac07c69a71ec2ec7432c573c8b0680f3b6e4bb3c30baaf845b2685840b133dbe260c332640904f0a523421cf911084e98dd18204e88271fbc1c35f18fb6"},
		},
		{
			image:    "{\"xkeys\":[{\"alias\":\"test4\",\"crypto\":{\"cipher\":\"aes-128-ctr\",\"cipherparams\":{\"iv\":\"45650919af7c61d907d8681ae8a3f8d0\"},\"ciphertext\":\"0d073355814763f3c4d0c049668d362a419af56ff193a1e65f1e867babaf3b8c9e91e539918314e7b78532216093f8bc2dd7712af579f7ff2b95ef0ecb8562a0\",\"kdf\":\"scrypt\",\"kdfparams\":{\"dklen\":32,\"n\":4096,\"p\":6,\"r\":8,\"salt\":\"d6a5ff1f5ba2d678db428cb2d387a8130f0223c9fe271dfdf02a3edca4608b30\"},\"mac\":\"889920bee3829a8b3aaccb8c6e8aa5f9dda7f986d72efa25c5777c88c1a86c17\"},\"id\":\"868047f4-6613-4abb-902a-4b2a288cd8c7\",\"type\":\"bytom_kd\",\"version\":1,\"xpub\":\"a57f2ac07c69a71ec2ec7432c573c8b0680f3b6e4bb3c30baaf845b2685840b133dbe260c332640904f0a523421cf911084e98dd18204e88271fbc1c35f18fb5\"}]}",
			wantErr:  ErrDuplicateKeyAlias,
			wantKeys: []string{"a57f2ac07c69a71ec2ec7432c573c8b0680f3b6e4bb3c30baaf845b2685840b133dbe260c332640904f0a523421cf911084e98dd18204e88271fbc1c35f18fb6"},
		},
		{
			image:    "{\"xkeys\":[{\"alias\":\"test4\",\"crypto\":{\"cipher\":\"aes-128-ctr\",\"cipherparams\":{\"iv\":\"45650919af7c61d907d8681ae8a3f8d0\"},\"ciphertext\":\"0d073355814763f3c4d0c049668d362a419af56ff193a1e65f1e867babaf3b8c9e91e539918314e7b78532216093f8bc2dd7712af579f7ff2b95ef0ecb8562a0\",\"kdf\":\"scrypt\",\"kdfparams\":{\"dklen\":32,\"n\":4096,\"p\":6,\"r\":8,\"salt\":\"d6a5ff1f5ba2d678db428cb2d387a8130f0223c9fe271dfdf02a3edca4608b30\"},\"mac\":\"889920bee3829a8b3aaccb8c6e8aa5f9dda7f986d72efa25c5777c88c1a86c17\"},\"id\":\"868047f4-6613-4abb-902a-4b2a288cd8c7\",\"type\":\"bytom_kd\",\"version\":1,\"xpub\":\"a57f2ac07c69a71ec2ec7432c573c8b0680f3b6e4bb3c30baaf845b2685840b133dbe260c332640904f0a523421cf911084e98dd18204e88271fbc1c35f18fb6\"},{\"alias\":\"test5\",\"crypto\":{\"cipher\":\"aes-128-ctr\",\"cipherparams\":{\"iv\":\"45650919af7c61d907d8681ae8a3f8d0\"},\"ciphertext\":\"0d073355814763f3c4d0c049668d362a419af56ff193a1e65f1e867babaf3b8c9e91e539918314e7b78532216093f8bc2dd7712af579f7ff2b95ef0ecb8562a0\",\"kdf\":\"scrypt\",\"kdfparams\":{\"dklen\":32,\"n\":4096,\"p\":6,\"r\":8,\"salt\":\"d6a5ff1f5ba2d678db428cb2d387a8130f0223c9fe271dfdf02a3edca4608b30\"},\"mac\":\"889920bee3829a8b3aaccb8c6e8aa5f9dda7f986d72efa25c5777c88c1a86c17\"},\"id\":\"868047f4-6613-4abb-902a-4b2a288cd8c7\",\"type\":\"bytom_kd\",\"version\":1,\"xpub\":\"a57f2ac07c69a71ec2ec7432c573c8b0680f3b6e4bb3c30baaf845b2685840b133dbe260c332640904f0a523421cf911084e98dd18204e88271fbc1c35f18fb7\"}]}",
			wantErr:  nil,
			wantKeys: []string{"a57f2ac07c69a71ec2ec7432c573c8b0680f3b6e4bb3c30baaf845b2685840b133dbe260c332640904f0a523421cf911084e98dd18204e88271fbc1c35f18fb6", "a57f2ac07c69a71ec2ec7432c573c8b0680f3b6e4bb3c30baaf845b2685840b133dbe260c332640904f0a523421cf911084e98dd18204e88271fbc1c35f18fb7"},
		},
	}

	for _, test := range testCases {
		keyImage := &KeyImage{}
		if err := json.Unmarshal([]byte(test.image), keyImage); err != nil {
			t.Fatal("unmarshal json error:", err)
		}

		if err := hsm.Restore(keyImage); err != test.wantErr {
			t.Errorf("error mismatch: have %v, want %v", err, test.wantErr)
		}

		if len(hsm.cache.keys()) != len(test.wantKeys) {
			t.Errorf("error key num: have %v, want %v", len(hsm.cache.keys()), len(test.wantKeys))
		}

		for _, key := range test.wantKeys {
			var xPub chainkd.XPub
			data, _ := hex.DecodeString(key)
			copy(xPub[:], data)

			if !hsm.cache.hasKey(xPub) {
				t.Errorf("error restore key: can't find key %v", key)
			}
		}
	}
}
