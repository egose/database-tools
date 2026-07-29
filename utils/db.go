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
	schemeIndex := strings.Index(uri, "://")
	if schemeIndex == -1 {
		return uri
	}

	authorityStart := schemeIndex + len("://")
	remainder := uri[authorityStart:]
	authorityEnd := strings.IndexAny(remainder, "/?")
	if authorityEnd == -1 {
		authorityEnd = len(remainder)
	}

	authority := remainder[:authorityEnd]
	atIndex := strings.LastIndex(authority, "@")
	if atIndex == -1 {
		return uri
	}

	return uri[:authorityStart] + authority[atIndex+1:] + remainder[authorityEnd:]
}
