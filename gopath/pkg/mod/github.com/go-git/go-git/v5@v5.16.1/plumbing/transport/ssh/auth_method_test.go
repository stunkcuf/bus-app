package ssh

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-billy/v5/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/testdata"

	. "gopkg.in/check.v1"
)

type (
	SuiteCommon struct{}

	mockKnownHosts         struct{}
	mockKnownHostsWithCert struct{}
)

func (mockKnownHosts) host() string { return "github.com" }
func (mockKnownHosts) knownHosts() []byte {
	return []byte(`github.com ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEAq2A7hRGmdnm9tUDbO9IDSwBK6TbQa+PXYPCPy6rbTrTtw7PHkccKrpp0yVhp5HdEIcKr6pLlVDBfOLX9QUsyCOV0wzfjIJNlGEYsdlLJizHhbn2mUjvSAHQqZETYP81eFzLQNnPHt4EVVUh7VfDESU84KezmD5QlWpXLmvU31/yMf+Se8xhHTvKSCZIFImWwoG6mbUoWf9nzpIoaSjB+weqqUUmpaaasXVal72J+UX2B+2RPW3RcT0eOzQgqlJL3RKrTJvdsjE3JEAvGq3lGHSZXy28G3skua2SmVi/w4yCE6gbODqnTWlg7+wC604ydGXA8VJiS5ap43JXiUFFAaQ==`)
}
func (mockKnownHosts) Network() string { return "tcp" }
func (mockKnownHosts) String() string  { return "github.com:22" }
func (mockKnownHosts) Algorithms() []string {
	return []string{ssh.KeyAlgoRSA, ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSASHA512}
}

func (mockKnownHostsWithCert) host() string { return "github.com" }
func (mockKnownHostsWithCert) knownHosts() []byte {
	return []byte(`@cert-authority github.com ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEAq2A7hRGmdnm9tUDbO9IDSwBK6TbQa+PXYPCPy6rbTrTtw7PHkccKrpp0yVhp5HdEIcKr6pLlVDBfOLX9QUsyCOV0wzfjIJNlGEYsdlLJizHhbn2mUjvSAHQqZETYP81eFzLQNnPHt4EVVUh7VfDESU84KezmD5QlWpXLmvU31/yMf+Se8xhHTvKSCZIFImWwoG6mbUoWf9nzpIoaSjB+weqqUUmpaaasXVal72J+UX2B+2RPW3RcT0eOzQgqlJL3RKrTJvdsjE3JEAvGq3lGHSZXy28G3skua2SmVi/w4yCE6gbODqnTWlg7+wC604ydGXA8VJiS5ap43JXiUFFAaQ==`)
}
func (mockKnownHostsWithCert) Network() string { return "tcp" }
func (mockKnownHostsWithCert) String() string  { return "github.com:22" }
func (mockKnownHostsWithCert) Algorithms() []string {
	return []string{ssh.CertAlgoRSASHA512v01, ssh.CertAlgoRSASHA256v01, ssh.CertAlgoRSAv01}
}

var _ = Suite(&SuiteCommon{})

func (s *SuiteCommon) TestKeyboardInteractiveName(c *C) {
	a := &KeyboardInteractive{
		User:      "test",
		Challenge: nil,
	}
	c.Assert(a.Name(), Equals, KeyboardInteractiveName)
}

func (s *SuiteCommon) TestKeyboardInteractiveString(c *C) {
	a := &KeyboardInteractive{
		User:      "test",
		Challenge: nil,
	}
	c.Assert(a.String(), Equals, fmt.Sprintf("user: test, name: %s", KeyboardInteractiveName))
}

func (s *SuiteCommon) TestPasswordName(c *C) {
	a := &Password{
		User:     "test",
		Password: "",
	}
	c.Assert(a.Name(), Equals, PasswordName)
}

func (s *SuiteCommon) TestPasswordString(c *C) {
	a := &Password{
		User:     "test",
		Password: "",
	}
	c.Assert(a.String(), Equals, fmt.Sprintf("user: test, name: %s", PasswordName))
}

func (s *SuiteCommon) TestPasswordCallbackName(c *C) {
	a := &PasswordCallback{
		User:     "test",
		Callback: nil,
	}
	c.Assert(a.Name(), Equals, PasswordCallbackName)
}

func (s *SuiteCommon) TestPasswordCallbackString(c *C) {
	a := &PasswordCallback{
		User:     "test",
		Callback: nil,
	}
	c.Assert(a.String(), Equals, fmt.Sprintf("user: test, name: %s", PasswordCallbackName))
}

func (s *SuiteCommon) TestPublicKeysName(c *C) {
	a := &PublicKeys{
		User:   "test",
		Signer: nil,
	}
	c.Assert(a.Name(), Equals, PublicKeysName)
}

func (s *SuiteCommon) TestPublicKeysString(c *C) {
	a := &PublicKeys{
		User:   "test",
		Signer: nil,
	}
	c.Assert(a.String(), Equals, fmt.Sprintf("user: test, name: %s", PublicKeysName))
}

func (s *SuiteCommon) TestPublicKeysCallbackName(c *C) {
	a := &PublicKeysCallback{
		User:     "test",
		Callback: nil,
	}
	c.Assert(a.Name(), Equals, PublicKeysCallbackName)
}

func (s *SuiteCommon) TestPublicKeysCallbackString(c *C) {
	a := &PublicKeysCallback{
		User:     "test",
		Callback: nil,
	}
	c.Assert(a.String(), Equals, fmt.Sprintf("user: test, name: %s", PublicKeysCallbackName))
}
func (s *SuiteCommon) TestNewSSHAgentAuth(c *C) {
	if runtime.GOOS == "js" {
		c.Skip("tcp connections are not available in wasm")
	}

	if os.Getenv("SSH_AUTH_SOCK") == "" {
		c.Skip("SSH_AUTH_SOCK or SSH_TEST_PRIVATE_KEY are required")
	}

	auth, err := NewSSHAgentAuth("foo")
	c.Assert(err, IsNil)
	c.Assert(auth, NotNil)
}

func (s *SuiteCommon) TestNewSSHAgentAuthNoAgent(c *C) {
	addr := os.Getenv("SSH_AUTH_SOCK")
	err := os.Unsetenv("SSH_AUTH_SOCK")
	c.Assert(err, IsNil)

	defer func() {
		err := os.Setenv("SSH_AUTH_SOCK", addr)
		c.Assert(err, IsNil)
	}()

	k, err := NewSSHAgentAuth("foo")
	c.Assert(k, IsNil)
	c.Assert(err, ErrorMatches, ".*SSH_AUTH_SOCK.*|.*SSH agent .* not detect.*")
}

func (*SuiteCommon) TestNewPublicKeys(c *C) {
	auth, err := NewPublicKeys("foo", testdata.PEMBytes["rsa"], "")
	c.Assert(err, IsNil)
	c.Assert(auth, NotNil)
}

func (*SuiteCommon) TestNewPublicKeysWithEncryptedPEM(c *C) {
	f := testdata.PEMEncryptedKeys[0]
	auth, err := NewPublicKeys("foo", f.PEMBytes, f.EncryptionKey)
	c.Assert(err, IsNil)
	c.Assert(auth, NotNil)
}

func (*SuiteCommon) TestNewPublicKeysWithEncryptedEd25519PEM(c *C) {
	f := testdata.PEMEncryptedKeys[2]
	auth, err := NewPublicKeys("foo", f.PEMBytes, f.EncryptionKey)
	c.Assert(err, IsNil)
	c.Assert(auth, NotNil)
}

func (*SuiteCommon) TestNewPublicKeysFromFile(c *C) {
	if runtime.GOOS == "js" {
		c.Skip("not available in wasm")
	}

	f, err := util.TempFile(osfs.Default, "", "ssh-test")
	c.Assert(err, IsNil)
	_, err = f.Write(testdata.PEMBytes["rsa"])
	c.Assert(err, IsNil)
	c.Assert(f.Close(), IsNil)
	defer osfs.Default.Remove(f.Name())

	auth, err := NewPublicKeysFromFile("foo", f.Name(), "")
	c.Assert(err, IsNil)
	c.Assert(auth, NotNil)
}

func (*SuiteCommon) TestNewPublicKeysWithInvalidPEM(c *C) {
	auth, err := NewPublicKeys("foo", []byte("bar"), "")
	c.Assert(err, NotNil)
	c.Assert(auth, IsNil)
}

func (*SuiteCommon) TestNewKnownHostsCallback(c *C) {
	if runtime.GOOS == "js" {
		c.Skip("not available in wasm")
	}

	var mock = mockKnownHosts{}

	f, err := util.TempFile(osfs.Default, "", "known-hosts")
	c.Assert(err, IsNil)

	_, err = f.Write(mock.knownHosts())
	c.Assert(err, IsNil)

	err = f.Close()
	c.Assert(err, IsNil)

	defer util.RemoveAll(osfs.Default, f.Name())

	f, err = osfs.Default.Open(f.Name())
	c.Assert(err, IsNil)

	defer f.Close()

	var hostKey ssh.PublicKey
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), " ")
		if len(fields) != 3 {
			continue
		}
		if strings.Contains(fields[0], mock.host()) {
			var err error
			hostKey, _, _, _, err = ssh.ParseAuthorizedKey(scanner.Bytes())
			if err != nil {
				c.Fatalf("error parsing %q: %v", fields[2], err)
			}
			break
		}
	}
	if hostKey == nil {
		c.Fatalf("no hostkey for %s", mock.host())
	}

	clb, err := NewKnownHostsCallback(f.Name())
	c.Assert(err, IsNil)

	err = clb(mock.String(), mock, hostKey)
	c.Assert(err, IsNil)
}

func (*SuiteCommon) TestNewKnownHostsDbWithoutCert(c *C) {
	if runtime.GOOS == "js" {
		c.Skip("not available in wasm")
	}

	var mock = mockKnownHosts{}

	f, err := util.TempFile(osfs.Default, "", "known-hosts")
	c.Assert(err, IsNil)

	_, err = f.Write(mock.knownHosts())
	c.Assert(err, IsNil)

	err = f.Close()
	c.Assert(err, IsNil)

	defer util.RemoveAll(osfs.Default, f.Name())

	f, err = osfs.Default.Open(f.Name())
	c.Assert(err, IsNil)

	defer f.Close()

	db, err := NewKnownHostsDb(f.Name())
	c.Assert(err, IsNil)

	algos := db.HostKeyAlgorithms(mock.String())
	c.Assert(algos, HasLen, len(mock.Algorithms()))

	for _, algorithm := range mock.Algorithms() {
		if !slices.Contains(algos, algorithm) {
			c.Error("algos does not contain ", algorithm)
		}
	}
}

func (*SuiteCommon) TestNewKnownHostsDbWithCert(c *C) {
	if runtime.GOOS == "js" {
		c.Skip("not available in wasm")
	}

	var mock = mockKnownHostsWithCert{}

	f, err := util.TempFile(osfs.Default, "", "known-hosts")
	c.Assert(err, IsNil)

	_, err = f.Write(mock.knownHosts())
	c.Assert(err, IsNil)

	err = f.Close()
	c.Assert(err, IsNil)

	defer util.RemoveAll(osfs.Default, f.Name())

	f, err = osfs.Default.Open(f.Name())
	c.Assert(err, IsNil)

	defer f.Close()

	db, err := NewKnownHostsDb(f.Name())
	c.Assert(err, IsNil)

	algos := db.HostKeyAlgorithms(mock.String())
	c.Assert(algos, HasLen, len(mock.Algorithms()))

	for _, algorithm := range mock.Algorithms() {
		if !slices.Contains(algos, algorithm) {
			c.Error("algos does not contain ", algorithm)
		}
	}
}

func TestHostKeyCallbackHelper(t *testing.T) {
	cb1 := ssh.FixedHostKey(nil)
	tests := []struct {
		name     string
		cb       ssh.HostKeyCallback
		algos    []string
		fallback func(files ...string) (ssh.HostKeyCallback, error)
		cc       *ssh.ClientConfig
		want     *ssh.ClientConfig
		wantErr  string
	}{
		{
			name: "keep existing callback if set",
			cb:   cb1,
			cc:   &ssh.ClientConfig{},
			want: &ssh.ClientConfig{
				HostKeyCallback: cb1,
			},
		},
		{
			name: "create new client config is one isn't provided",
			cb:   cb1,
			cc:   nil,
			want: &ssh.ClientConfig{
				HostKeyCallback: cb1,
			},
		},
		{
			name:  "respect pre-set algos",
			cb:    cb1,
			algos: []string{"foo"},
			cc:    &ssh.ClientConfig{},
			want: &ssh.ClientConfig{
				HostKeyCallback:   cb1,
				HostKeyAlgorithms: []string{"foo"},
			},
		},
		{
			name: "no callback is set, call fallback",
			cc:   &ssh.ClientConfig{},
			fallback: func(files ...string) (ssh.HostKeyCallback, error) {
				return cb1, nil
			},
			want: &ssh.ClientConfig{
				HostKeyCallback: cb1,
			},
		},
		{
			name: "no callback is set with nil client config",
			fallback: func(files ...string) (ssh.HostKeyCallback, error) {
				return cb1, nil
			},
			want: &ssh.ClientConfig{
				HostKeyCallback: cb1,
			},
		},
		{
			name:  "algos with no callback, call fallback",
			algos: []string{"bar"},
			cc:    &ssh.ClientConfig{},
			fallback: func(files ...string) (ssh.HostKeyCallback, error) {
				return cb1, nil
			},
			want: &ssh.ClientConfig{
				HostKeyCallback:   cb1,
				HostKeyAlgorithms: []string{"bar"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := HostKeyCallbackHelper{
				HostKeyCallback:   tc.cb,
				HostKeyAlgorithms: tc.algos,
				fallback:          tc.fallback,
			}

			got, gotErr := helper.SetHostKeyCallback(tc.cc)

			if tc.wantErr == "" {
				require.NoError(t, gotErr)
				require.NotNil(t, got)

				wantFunc := runtime.FuncForPC(reflect.ValueOf(tc.want.HostKeyCallback).Pointer()).Name()
				gotFunc := runtime.FuncForPC(reflect.ValueOf(got.HostKeyCallback).Pointer()).Name()
				assert.Equal(t, wantFunc, gotFunc)

				assert.Equal(t, tc.want.HostKeyAlgorithms, got.HostKeyAlgorithms)
			} else {
				assert.ErrorContains(t, gotErr, tc.wantErr)
				assert.Nil(t, got)
			}
		})
	}
}
