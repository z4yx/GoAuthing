package libauth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/juju/loggo"
)

const LogUriBase = "https://auth%d.tsinghua.edu.cn/cgi-bin/srun_portal"
const OnlineCheckUriBase = "https://auth%d.tsinghua.edu.cn/srun_portal_pc.php"
const UsageUriBase = "https://auth%d.tsinghua.edu.cn/rad_user_info.php"
const ChallengeUriBase = "https://auth%d.tsinghua.edu.cn/cgi-bin/get_challenge"

var logger = loggo.GetLogger("libauth")

func extractJSONFromJSONP(jsonp, callbackName string) (string, error) {
	l := len(callbackName)
	if len(jsonp) < l+2 {
		return "", errors.New("JSONP string too short")
	}
	if jsonp[:l] != callbackName || jsonp[l] != '(' || jsonp[len(jsonp)-1] != ')' {
		return "", errors.New("Invalid format")
	}
	return jsonp[l+1 : len(jsonp)-1], nil
}

func buildChallengeParams(username string) url.Values {

	challParams := url.Values{
		"username":     []string{username},
		"ip":           []string{""},
		"double_stack": []string{"1"},
	}

	return challParams
}

func buildLoginParams(username, password, token string, logout bool) (loginParams url.Values, err error) {
	ip := ""
	acID := "1"
	//Don't care
	hmd5 := "00000000000000000000000000000000" //md5sum(password)

	action := "login"
	if logout {
		action = "logout"
	}
	rawInfo := map[string]string{
		"username": username,
		"password": password,
		"ip":       ip,
		"acid":     acID,
		"enc_ver":  "s" + "run" + "_bx1",
	}
	infoJSON, _ := json.Marshal(rawInfo)
	// fmt.Printf("infoJSON: %s\n", infoJSON)
	loginParams = url.Values{
		"action":       []string{action},
		"ac_id":        []string{acID},
		"n":            []string{"200"},
		"type":         []string{"1"},
		"ip":           []string{ip},
		"double_stack": []string{"1"},
		"username":     []string{username},
		"password":     []string{"{MD5}" + hmd5},
	}
	encoded := XEncode(string(infoJSON), token)
	if encoded == nil {
		err = errors.New("XEncode failed")
		return
	}
	loginParams.Add("info", "{SRBX1}"+QuirkBase64Encode(*encoded))
	// fmt.Printf("chksum(raw): %v\n", token+username+token + hmd5+token+acID+token+ip+token+loginParams.Get("n")+token+loginParams.Get("type")+token+loginParams.Get("info"))
	loginParams.Add("chksum", sha1sum(token+username+token+hmd5+token+acID+token+ip+token+loginParams.Get("n")+token+loginParams.Get("type")+token+loginParams.Get("info")))
	// fmt.Printf("loginParams: %v\n", loginParams)
	return
}

func GetJSON(baseUrl string, params url.Values) (string, error) {
	const CB = "C_a_l_l_b_a_c_k"
	params.Set("callback", CB)
	var netClient = &http.Client{
		Timeout: time.Second * 2,
	}
	url := baseUrl + "?" + params.Encode()
	logger.Debugf("GET \"%s\"\n", url)
	resp, err := netClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return extractJSONFromJSONP(string(body), CB)
}

func IsOnline(proto int) (online bool, err error) {
	var netClient = &http.Client{
		Timeout: time.Second * 2,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			logger.Debugf("REDIRECT \"%v\"\n", req.URL)
			if strings.Index(req.URL.Path, "succeed_wired.php") != -1 {
				online = true
			}
			return nil
		},
	}
	online = false
	params := url.Values{
		"ac_id": []string{"1"},
	}
	url := fmt.Sprintf(OnlineCheckUriBase, proto) + "?" + params.Encode()
	logger.Debugf("GET \"%s\"\n", url)
	resp, err := netClient.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	return
}

func LoginLogout(username, password string, proto int, logout bool) (success bool, err error) {
	success = false
	logger.Debugf("Getting challenge...\n")
	body, err := GetJSON(fmt.Sprintf(ChallengeUriBase, proto), buildChallengeParams(username))
	if err != nil {
		return
	}
	logger.Debugf("Challenge response: %v\n", body)

	var challResp map[string]interface{}
	err = json.Unmarshal([]byte(body), &challResp)
	if err != nil {
		return
	}
	res, valid := challResp["res"].(string)
	if !valid || res != "ok" {
		err = errors.New("Failed to get challenge: " + res)
		return
	}
	token, valid := challResp["challenge"].(string)
	if !valid {
		err = errors.New("No challenge field")
		return
	}

	loginParams, err := buildLoginParams(username, password, token, logout)
	if err != nil {
		return
	}
	logger.Debugf("Sending login request...\n")
	body, err = GetJSON(fmt.Sprintf(LogUriBase, proto), loginParams)
	if err != nil {
		return
	}
	logger.Debugf("Login response: %v\n", body)
	var loginResp map[string]interface{}
	err = json.Unmarshal([]byte(body), &loginResp)
	if err != nil {
		return
	}
	res, valid = loginResp["error"].(string)
	if !valid {
		err = errors.New("No error field")
		return
	}

	if res == "ok" {
		success = true
		err = nil
	} else {
		ecode, _ := loginResp["ecode"].(string)
		if strerr, exist := PortalError[ecode]; exist {
			err = errors.New(strerr)
		} else {
			err = errors.New(res)
		}
	}

	return
}
