package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Response is of type APIGatewayProxyResponse since we're leveraging the
// AWS Lambda Proxy Request functionality (default behavior)
//
// https://serverless.com/framework/docs/providers/aws/events/apigateway/#lambda-proxy-integration
type Response events.APIGatewayProxyResponse
type Request events.APIGatewayProxyRequest

func fetchGameList(steamId64 string) (string, error) {
	apikey := os.Getenv("SteamWebApiKey")
	url := fmt.Sprintf("http://api.steampowered.com/IPlayerService/GetOwnedGames/v0001/?key=%s&steamid=%s&format=json", apikey, steamId64)
	res, err := http.Get(url)

	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		return "", fmt.Errorf("Steam API response code for fetching game list: %v", res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(body), err
}

func createResponse(status int, body string) Response {
	return Response{
		StatusCode:      status,
		IsBase64Encoded: false,
		Body:            body,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "http://localhost:8080",
			"X-Content-Type-Options":      "nosniff",
		},
	}
}

func addProfileToJson(inputJson string, profile Profile) string {
	out := map[string]interface{}{}
	json.Unmarshal([]byte(inputJson), &out)
	out["SteamID"] = profile.SteamID
	out["SteamID64"] = profile.SteamID64
	out["AvatarIcon"] = profile.AvatarIcon

	outputJson, _ := json.Marshal(out)
	return string(outputJson)
}

func GetGameList(ctx context.Context, request Request) (Response, error) {
	user := request.QueryStringParameters["user"]
	if user == "" {
		return createResponse(418, "No user given"), nil
	}

	profile, err := GetProfile(user)
	if err != nil {
		return createResponse(418, err.Error()), nil
	}

	body, err := fetchGameList(profile.SteamID64)
	if err != nil {
		return createResponse(418, err.Error()), nil
	}

	newJson := addProfileToJson(body, profile)

	return createResponse(200, newJson), nil
}

func main() {
	lambda.Start(GetGameList)
}
