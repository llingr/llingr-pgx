package connect

import (
	"net"
	"net/url"
	"strconv"
	"strings"
	"unicode"
)

// DSN renders the config as a libpq keyword/value string.
//
// The rendered string embeds the password: treat it as a credential, not as log
// output.
func (b *ConnectionBuilder) DSN() string {
	pairs := b.keyValues()
	parts := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		parts = append(parts, pair[0]+"="+quoteKeywordValue(pair[1]))
	}
	return strings.Join(parts, " ")
}

// PSQL renders the config as a libpq connection URI ("postgres://…"), percent-
// encoding the userinfo and query values. dbname becomes the path; everything
// else (sslmode, extra params, pool_*) becomes query parameters.
//
// The rendered string embeds the password: treat it as a credential, not as log
// output.
//
// For multi-host use DSN or direct connection string : PSQL URL builder does
// not currently support this.
func (b *ConnectionBuilder) PSQL() string {
	uri := url.URL{Scheme: SchemePostgres}

	if b.user != "" {
		if b.password != "" {
			uri.User = url.UserPassword(b.user, b.password)
		} else {
			uri.User = url.User(b.user)
		}
	}

	// A unix-socket host is an absolute path, which is not a valid URL.
	// libpq expects it in the host query parameter instead, leaving the authority
	// empty (postgres://user@/dbname?host=/var/run/postgresql). A zoned IPv6 host
	// (fe80::1%eth0) with no port also goes in the query parameter: pgx cannot
	// parse the bracketed form without a port.
	hostInQuery := strings.HasPrefix(b.host, "/") ||
		(b.port == 0 && strings.Contains(b.host, "%"))
	if !hostInQuery {
		switch {
		case b.host != "" && b.port != 0:
			uri.Host = net.JoinHostPort(b.host, strconv.FormatUint(uint64(b.port), 10))
		case b.host != "":
			// A bare IPv6 literal must be bracketed to form a valid URL authority
			// (net.JoinHostPort does this in the with-port branch).
			if strings.Contains(b.host, ":") {
				uri.Host = "[" + b.host + "]"
			} else {
				uri.Host = b.host
			}
		case b.port != 0:
			uri.Host = net.JoinHostPort("", strconv.FormatUint(uint64(b.port), 10))
		}
	}

	if b.database != "" {
		uri.Path = "/" + b.database
	}

	// Query carries everything not already placed in the userinfo, authority, or
	// path. For a TCP host that excludes host/port (they are the authority); for a
	// socket host they belong here instead.
	query := make([]string, 0)
	for _, pair := range b.keyValues() {
		switch pair[0] {
		case ParamUser, ParamDatabase:
			continue
		case ParamPassword:
			// The password sits in the userinfo only when a user is set. Without
			// one it travels as a query parameter (libpq accepts any keyword
			// there) rather than being silently dropped.
			if b.user != "" {
				continue
			}
		case ParamHost, ParamPort:
			if !hostInQuery {
				continue
			}
		}
		query = append(query, url.QueryEscape(pair[0])+"="+url.QueryEscape(pair[1]))
	}
	uri.RawQuery = strings.Join(query, "&")

	return uri.String()
}

// keyValues returns the connection settings as ordered (key, value) pairs in libpq
// keyword spelling. Both renderers share this so DSN and PSQL stay aligned.
func (b *ConnectionBuilder) keyValues() [][2]string {
	pairs := make([][2]string, 0, 8+len(b.paramKeys))
	add := func(key, value string) {
		pairs = append(pairs, [2]string{key, value})
	}
	if b.host != "" {
		add(ParamHost, b.host)
	}
	if b.port != 0 {
		add(ParamPort, strconv.FormatUint(uint64(b.port), 10))
	}
	if b.user != "" {
		add(ParamUser, b.user)
	}
	if b.password != "" {
		add(ParamPassword, b.password)
	}
	if b.database != "" {
		add(ParamDatabase, b.database)
	}
	if b.sslmode != "" {
		add(ParamSSLMode, b.sslmode)
	}
	if b.channelBinding != "" {
		add(ParamChannelBinding, b.channelBinding)
	}
	for _, key := range b.paramKeys {
		add(key, b.params[key])
	}
	if b.maxConns != 0 {
		add(ParamPoolMaxConns, strconv.FormatInt(int64(b.maxConns), 10))
	}
	if b.minConns != 0 {
		add(ParamPoolMinConns, strconv.FormatInt(int64(b.minConns), 10))
	}
	if b.minIdleConns != 0 {
		add(ParamPoolMinIdleConns, strconv.FormatInt(int64(b.minIdleConns), 10))
	}
	if b.maxConnLifetime != 0 {
		add(ParamPoolMaxConnLifetime, b.maxConnLifetime.String())
	}
	if b.maxConnLifetimeJitter != 0 {
		add(ParamPoolMaxConnLifetimeJitter, b.maxConnLifetimeJitter.String())
	}
	if b.maxConnIdleTime != 0 {
		add(ParamPoolMaxConnIdleTime, b.maxConnIdleTime.String())
	}
	if b.healthCheckPeriod != 0 {
		add(ParamPoolHealthCheckPeriod, b.healthCheckPeriod.String())
	}
	return pairs
}

// quoteKeywordValue applies libpq keyword/value quoting: empty values and any value
// containing whitespace (space, tab, newline, ...), a single quote, or a backslash
// are wrapped in single quotes with backslash and single quote escaped. Quoting on
// all whitespace matters: an unquoted tab or newline breaks keyword/value parsing
// outright, whereas the quoted form round-trips byte for byte.
func quoteKeywordValue(value string) string {
	if value == "" || strings.ContainsAny(value, `'\`) || strings.ContainsFunc(value, unicode.IsSpace) {
		escaped := strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(value)
		return "'" + escaped + "'"
	}
	return value
}
