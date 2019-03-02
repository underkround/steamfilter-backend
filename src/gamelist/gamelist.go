package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
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
		log.Fatal(err)
		return "", err
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
		return "", fmt.Errorf("Steam API response code: %s", res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	return string(body), err
}

func createGameListUrl(user string) (string, error) {
	if user == "" {
		log.Fatalf("Invalid user: %s", user)
		return "", fmt.Errorf("Invalid user")
	}

	// TODO: Different url types
	url := fmt.Sprintf("https://steamcommunity.com/id/%s?xml=1", user)
	return url, nil
}

func GetGameList(ctx context.Context, request Request) (Response, error) {
	user := request.QueryStringParameters["user"]
	url, err := createGameListUrl(user)
	status := 200
	var body string

	if err != nil {
		status = 403
		body = err.Error()
	} else {
		body, err = fetchGameList(url)

		if err != nil {
			status = 403
			body = err.Error()
		}
	}

	resp := Response{
		StatusCode:      status,
		IsBase64Encoded: false,
		Body:            body,
		//		Headers: map[string]string{
		//			"Content-Type":           "application/json",
		//			"X-MyCompany-Func-Reply": "hello-handler",
		//		},
	}

	return resp, nil
}

func main() {
	lambda.Start(GetGameList)
}
