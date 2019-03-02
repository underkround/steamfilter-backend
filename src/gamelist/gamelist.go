package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Response is of type APIGatewayProxyResponse since we're leveraging the
// AWS Lambda Proxy Request functionality (default behavior)
//
// https://serverless.com/framework/docs/providers/aws/events/apigateway/#lambda-proxy-integration
type Response events.APIGatewayProxyResponse
type Request events.APIGatewayProxyRequest

func fetchGameList(url string) (string, error) {
	res, err := http.Get(url)

	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		return "", fmt.Errorf("Steam API response code: %s", res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(body), err
}

func createGameListUrl(user string) (string, error) {
	fmt.Printf("User: %s\n", user)
	if user == "" {
		return "", fmt.Errorf("No user given")
	}

	// TODO: Different url types
	//url := fmt.Sprintf("https://steamcommunity.com/id/%s/games/?tab=all&xml=1", user)
	url := fmt.Sprintf("http://api.steampowered.com/IPlayerService/GetOwnedGames/v0001/?key=668C56808461A02FC1E7F600464FC48D&steamid=%s&format=json", user)
	return url, nil
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

func GetGameList(ctx context.Context, request Request) (Response, error) {
	user := request.QueryStringParameters["user"]
	url, err := createGameListUrl(user)

	if err != nil {
		return createResponse(418, err.Error()), nil
	}

	body, err := fetchGameList(url)
	if err != nil {
		return createResponse(418, err.Error()), nil
	}

	return createResponse(200, body), nil
}

func main() {
	lambda.Start(GetGameList)
}
