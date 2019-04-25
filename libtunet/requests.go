package libtunet

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/juju/loggo"
)

var logger = loggo.GetLogger("libtunet")

func md5sum(input string) string {
	h := md5.New()
	io.WriteString(h, input)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func LoginLogout(username, password string, logout bool) (success bool, err error) {
	md5pwd := md5sum(password)
	var action string
	if logout {
		action = "Logout"
	} else {
		action = "Login"
	}
	loginParams := url.Values{
		"action": []string{"logout"},
	}
	if !logout {
		loginParams = url.Values{
			"action":   []string{"login"},
			"ac_id":    []string{"1"},
			"username": []string{username},
			"password": []string{"{MD5_HEX}" + md5pwd},
		}
	}
	netClient := &http.Client{
		Timeout: time.Second * 2,
	}
	url := "http://net.tsinghua.edu.cn/do_login.php?" + loginParams.Encode()
	logger.Debugf("Sending %s request...\n", action)
	logger.Debugf("GET \"%s\"\n", url)
	resp, err := netClient.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	bodyB, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	body := string(bodyB)
	logger.Debugf("%s response: %v\n", action, body)
	if body == fmt.Sprintf("%s is successful.", action) {
		return true, err
	} else {
		err = errors.New(body)
		return false, err
	}
}
