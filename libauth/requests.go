package libauth

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/juju/loggo"
)

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

func buildChallengeParams(username string, anotherIP string) url.Values {

	challParams := url.Values{
		"username":     []string{username},
		"ip":           []string{anotherIP},
		"double_stack": []string{"1"},
	}

	return challParams
}

func buildLoginParams(username, password, token string, logout bool, anotherIP string, acID string) (loginParams url.Values, err error) {
	ip := anotherIP
	//Required by wireless network only
	hmd5 := fmt.Sprintf("%032x", md5.Sum([]byte(password)))

	action := "login"
	rawInfo := map[string]string{
		"username": username,
		"password": password,
		"ip":       ip,
		"acid":     acID,
		"enc_ver":  "s" + "run" + "_bx1",
	}
	if logout {
		action = "logout"
		delete(rawInfo, "password")
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
	}
	if !logout {
		loginParams.Add("password", "{MD5}"+hmd5)
	}
	encoded := XEncode(string(infoJSON), token)
	if encoded == nil {
		err = errors.New("XEncode failed")
		return
	}
	loginParams.Add("info", "{SRBX1}"+QuirkBase64Encode(*encoded))
	// fmt.Printf("chksum(raw): %v\n", token+username+token + hmd5+token+acID+token+ip+token+loginParams.Get("n")+token+loginParams.Get("type")+token+loginParams.Get("info"))
	if logout {
		loginParams.Add("chksum", sha1sum(token+username+token+acID+token+ip+token+loginParams.Get("n")+token+loginParams.Get("type")+token+loginParams.Get("info")))
	} else {
		loginParams.Add("chksum", sha1sum(token+username+token+hmd5+token+acID+token+ip+token+loginParams.Get("n")+token+loginParams.Get("type")+token+loginParams.Get("info")))
	}
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

func IsOnline(host *UrlProvider, acID string) (online bool, err error, username string) {
	logger.Debugf("Check if online\n")
	var netClient = &http.Client{
		Timeout: time.Second * 2,
	}
	online = false
	params := url.Values{
		"ac_id": []string{acID},
	}
	uri := host.OnlineCheckUriBase() + "?" + params.Encode()
	logger.Debugf("GET \"%s\"\n", uri)
	resp, err := netClient.Get(uri)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// find public ip from response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	regexMatchIP := regexp.MustCompile(`ip\s+:\s"([0-9.]+)"`)
	matches := regexMatchIP.FindStringSubmatch(string(body))
	if len(matches) < 2 {
		err = errors.New("ip not found")
		return
	}
	ip := matches[1]
	logger.Debugf("ip=%s\n", ip)

	// Get user info
	logger.Debugf("Get user info\n")
	params = url.Values{
		"ip":           []string{ip},
	}
	info, err := GetJSON(host.UserInfoUriBase(), params)

	var infoResp map[string]interface{}
	err = json.Unmarshal([]byte(info), &infoResp)
	logger.Debugf("Get user info %v\n", infoResp)
	if err != nil {
		return
	}

	res, valid := infoResp["error"].(string)
	if valid && res == "ok" {
		online = true
		logger.Debugf("User is online\n")
	}

	res, valid = infoResp["user_name"].(string)
	if valid {
		username = res
		logger.Debugf("User name is \"%s\"\n", username)
	}

	return
}

func GetNasID(IP, user, password string) (nasID string, err error) {
	err = errors.New("Not implemented for usereg 2025")
	return
}

func GetAcID(V6 bool) (acID string, err error) {
	logger.Debugf("Get AC ID\n")
	var netClient = &http.Client{
		Timeout: time.Second * 2,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			logger.Debugf("REDIRECT \"%v\"\n", req.URL)
			return errors.New("should not redirect")
		},
	}
	acID = ""
	var resp *http.Response
	var body []byte
	url := "http://login.tsinghua.edu.cn/index_1.html"
	logger.Debugf("GET \"%s\"\n", url)
	resp, err = netClient.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	regexMatchAcID := regexp.MustCompile(`(ac_id=|index_)([0-9]+)`)
	matches := regexMatchAcID.FindStringSubmatch(string(body))
	if len(matches) < 3 {
		err = errors.New("ac_id not found")
		return
	}
	acID = matches[2]
	logger.Debugf("ac_id=%s\n", acID)
	return
}

func LoginLogout(username, password string, host *UrlProvider, logout bool, anotherIP string, acID string) (err error) {
	logger.Debugf("Getting challenge...\n")
	body, err := GetJSON(host.ChallengeUriBase(), buildChallengeParams(username, anotherIP))
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

	loginParams, err := buildLoginParams(username, password, token, logout, anotherIP, acID)
	if err != nil {
		return
	}
	logger.Debugf("Sending login request...\n")
	body, err = GetJSON(host.LoginUriBase(), loginParams)
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
