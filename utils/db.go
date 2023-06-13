package utils

import (
	"net/url"
	"strings"
)

func parseMongoDBURI(uri string) (string, string, string, string, string, string, error) {
	parsedURI, err := url.Parse(uri)
	if err != nil {
		return "", "", "", "", "", "", err
	}

	username := ""
	password := ""
	if parsedURI.User != nil {
		username = parsedURI.User.Username()
		password, _ = parsedURI.User.Password()
	}

	scheme := parsedURI.Scheme
	host := parsedURI.Hostname()
	port := parsedURI.Port()
	database := strings.TrimPrefix(parsedURI.Path, "/")

	return scheme, host, port, username, password, database, nil
}

func generateMongoDBURI(scheme, host, port, username, password string) string {
	conn := ""

	if scheme != "" {
		conn += scheme + "://"
	} else {
		conn += "mongodb://"
	}

	if username != "" {
		conn += username
		if password != "" {
			conn += ":" + url.QueryEscape(password)
		}
		conn += "@"
	}

	if host != "" {
		conn += host
	} else {
		conn += "localhost"
	}

	if port != "" {
		conn += ":" + port
	}

	return conn
}

func PruneMongoDBURI(uri string) string {
	scheme, host, port, username, password, _, _ := parseMongoDBURI(uri)
	return generateMongoDBURI(scheme, host, port, username, password)
}
