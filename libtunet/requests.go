package libtunet

import (
	//"crypto/md5"
	//"encoding/json"
	//"errors"
	//"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	//"regexp"
	//"strings"
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
	success = false
	md5pwd := md5sum(password)
	loginParams := url.Values{
		"action":   []string{"login"},
		"ac_id":    []string{"1"},
		"username": []string{username},
		"password": []string{"{MD5_HEX}" + md5pwd},
	}
	netClient := &http.Client{
		Timeout: time.Second * 2,
	}
	url := "http://net.tsinghua.edu.cn/do_login.php?" + loginParams.Encode()
	logger.Debugf("Sending login request...\n")
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
	logger.Debugf("Login response: %v\n", body)
	return true, err
}
