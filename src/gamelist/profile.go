package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

type Profile struct {
	SteamID    string `xml:"steamID"`
	SteamID64  string `xml:"steamID64"`
	AvatarIcon string `xml:"avatarFull"`
}

func GetProfileName(s string) (string, bool) {
	if strings.HasPrefix(s, "http") {
		vanityRegex, _ := regexp.Compile("https://steamcommunity.com/id/([^/?]+)") // TODO don't compile every time
		a := vanityRegex.FindStringSubmatch(s)
		if len(a) == 0 {
			idRegex, _ := regexp.Compile("https://steamcommunity.com/profiles/([^/?]+)")
			b := idRegex.FindStringSubmatch(s)
			if len(b) == 0 {
				return s, true
			}
			return b[1], false
		}
		return a[1], true
	}

	idRegex, _ := regexp.Compile("([0-9]{17})")
	a := idRegex.FindStringSubmatch(s)
	if len(a) == 0 {
		return s, true
	}

	return s, false
}

func getProfileUrl(name string, isVanity bool) string {
	if isVanity {
		return fmt.Sprintf("https://steamcommunity.com/id/%s?xml=1", name)
	} else {
		return fmt.Sprintf("https://steamcommunity.com/profiles/%s?xml=1", name)
	}
}

func parseProfile(data []byte) (Profile, error) {
	var profile Profile

	xml.Unmarshal(data, &profile)
	return profile, nil
}

func fetchProfile(url string) (Profile, error) {
	var profile Profile

	res, err := http.Get(url)
	if err != nil {
		return profile, err
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		return profile, fmt.Errorf("Steam API response code for fetching profile: %v (url: %v)", res.StatusCode, url)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return profile, err
	}

	profile, err = parseProfile(body)
	return profile, err
}

func GetProfile(user string) (Profile, error) {
	var profile Profile
	if user == "" {
		return profile, fmt.Errorf("No profile name given")
	}

	profileName, isVanity := GetProfileName(user)
	url := getProfileUrl(profileName, isVanity)
	profile, err := fetchProfile(url)
	return profile, err
}
