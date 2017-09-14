package types

import (
	"encoding"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/libkv/store"
	"github.com/ryanuber/go-glob"
)

// Backend holds backend configuration.
type Backend struct {
	Servers        map[string]Server `json:"servers,omitempty"`
	CircuitBreaker *CircuitBreaker   `json:"circuitBreaker,omitempty"`
	LoadBalancer   *LoadBalancer     `json:"loadBalancer,omitempty"`
	MaxConn        *MaxConn          `json:"maxConn,omitempty"`
	HealthCheck    *HealthCheck      `json:"healthCheck,omitempty"`
	AuditTap       *AuditSink        `json:"auditTap,omitempty"`
}

// MaxConn holds maximum connection configuration
type MaxConn struct {
	Amount        int64  `json:"amount,omitempty"`
	ExtractorFunc string `json:"extractorFunc,omitempty"`
}

// LoadBalancer holds load balancing configuration.
type LoadBalancer struct {
	Method string `json:"method,omitempty"`
	Sticky bool   `json:"sticky,omitempty"`
}

// CircuitBreaker holds circuit breaker configuration.
type CircuitBreaker struct {
	Expression string `json:"expression,omitempty"`
}

// HealthCheck holds HealthCheck configuration
type HealthCheck struct {
	Path     string `json:"path,omitempty"`
	Interval string `json:"interval,omitempty"`
}

// Server holds server configuration.
type Server struct {
	URL    string `json:"url,omitempty"`
	Weight int    `json:"weight"`
}

// Route holds route configuration.
type Route struct {
	Rule string `json:"rule,omitempty"`
}

// Headers holds the custom header configuration
type Headers struct {
	CustomRequestHeaders    map[string]string `json:"customRequestHeaders,omitempty"`
	CustomResponseHeaders   map[string]string `json:"customResponseHeaders,omitempty"`
	AllowedHosts            []string          `json:"allowedHosts,omitempty"`
	HostsProxyHeaders       []string          `json:"hostsProxyHeaders,omitempty"`
	SSLRedirect             bool              `json:"sslRedirect,omitempty"`
	SSLTemporaryRedirect    bool              `json:"sslTemporaryRedirect,omitempty"`
	SSLHost                 string            `json:"sslHost,omitempty"`
	SSLProxyHeaders         map[string]string `json:"sslProxyHeaders,omitempty"`
	STSSeconds              int64             `json:"stsSeconds,omitempty"`
	STSIncludeSubdomains    bool              `json:"stsIncludeSubdomains,omitempty"`
	STSPreload              bool              `json:"stsPreload,omitempty"`
	ForceSTSHeader          bool              `json:"forceSTSHeader,omitempty"`
	FrameDeny               bool              `json:"frameDeny,omitempty"`
	CustomFrameOptionsValue string            `json:"customFrameOptionsValue,omitempty"`
	ContentTypeNosniff      bool              `json:"contentTypeNosniff,omitempty"`
	BrowserXSSFilter        bool              `json:"browserXssFilter,omitempty"`
	ContentSecurityPolicy   string            `json:"contentSecurityPolicy,omitempty"`
	PublicKey               string            `json:"publicKey,omitempty"`
	ReferrerPolicy          string            `json:"referrerPolicy,omitempty"`
	IsDevelopment           bool              `json:"isDevelopment,omitempty"`
}

// HasCustomHeadersDefined checks to see if any of the custom header elements have been set
func (h Headers) HasCustomHeadersDefined() bool {
	return len(h.CustomResponseHeaders) != 0 ||
		len(h.CustomRequestHeaders) != 0
}

// HasSecureHeadersDefined checks to see if any of the secure header elements have been set
func (h Headers) HasSecureHeadersDefined() bool {
	return len(h.AllowedHosts) != 0 ||
		len(h.HostsProxyHeaders) != 0 ||
		h.SSLRedirect ||
		h.SSLTemporaryRedirect ||
		h.SSLHost != "" ||
		len(h.SSLProxyHeaders) != 0 ||
		h.STSSeconds != 0 ||
		h.STSIncludeSubdomains ||
		h.STSPreload ||
		h.ForceSTSHeader ||
		h.FrameDeny ||
		h.CustomFrameOptionsValue != "" ||
		h.ContentTypeNosniff ||
		h.BrowserXSSFilter ||
		h.ContentSecurityPolicy != "" ||
		h.PublicKey != "" ||
		h.ReferrerPolicy != "" ||
		h.IsDevelopment
}

// Frontend holds frontend configuration.
type Frontend struct {
	EntryPoints          []string         `json:"entryPoints,omitempty"`
	Backend              string           `json:"backend,omitempty"`
	Routes               map[string]Route `json:"routes,omitempty"`
	PassHostHeader       bool             `json:"passHostHeader,omitempty"`
	PassTLSCert          bool             `json:"passTLSCert,omitempty"`
	Priority             int              `json:"priority"`
	BasicAuth            []string         `json:"basicAuth"`
	WhitelistSourceRange []string         `json:"whitelistSourceRange,omitempty"`
	Headers              Headers          `json:"headers,omitempty"`
}

// LoadBalancerMethod holds the method of load balancing to use.
type LoadBalancerMethod uint8

const (
	// Wrr (default) = Weighted Round Robin
	Wrr LoadBalancerMethod = iota
	// Drr = Dynamic Round Robin
	Drr
)

var loadBalancerMethodNames = []string{
	"Wrr",
	"Drr",
}

// NewLoadBalancerMethod create a new LoadBalancerMethod from a given LoadBalancer.
func NewLoadBalancerMethod(loadBalancer *LoadBalancer) (LoadBalancerMethod, error) {
	var method string
	if loadBalancer != nil {
		method = loadBalancer.Method
		for i, name := range loadBalancerMethodNames {
			if strings.EqualFold(name, method) {
				return LoadBalancerMethod(i), nil
			}
		}
	}
	return Wrr, fmt.Errorf("invalid load-balancing method '%s'", method)
}

// Configuration of a provider.
type Configuration struct {
	Backends  map[string]*Backend  `json:"backends,omitempty"`
	Frontends map[string]*Frontend `json:"frontends,omitempty"`
}

// ConfigMessage hold configuration information exchanged between parts of traefik.
type ConfigMessage struct {
	ProviderName  string
	Configuration *Configuration
}

// Constraint hold a parsed constraint expresssion
type Constraint struct {
	Key string
	// MustMatch is true if operator is "==" or false if operator is "!="
	MustMatch bool
	// TODO: support regex
	Regex string
}

// NewConstraint receive a string and return a *Constraint, after checking syntax and parsing the constraint expression
func NewConstraint(exp string) (*Constraint, error) {
	sep := ""
	constraint := &Constraint{}

	if strings.Contains(exp, "==") {
		sep = "=="
		constraint.MustMatch = true
	} else if strings.Contains(exp, "!=") {
		sep = "!="
		constraint.MustMatch = false
	} else {
		return nil, errors.New("Constraint expression missing valid operator: '==' or '!='")
	}

	kv := strings.SplitN(exp, sep, 2)
	if len(kv) == 2 {
		// At the moment, it only supports tags
		if kv[0] != "tag" {
			return nil, errors.New("Constraint must be tag-based. Syntax: tag==us-*")
		}

		constraint.Key = kv[0]
		constraint.Regex = kv[1]
		return constraint, nil
	}

	return nil, errors.New("Incorrect constraint expression: " + exp)
}

func (c *Constraint) String() string {
	if c.MustMatch {
		return c.Key + "==" + c.Regex
	}
	return c.Key + "!=" + c.Regex
}

var _ encoding.TextUnmarshaler = (*Constraint)(nil)

// UnmarshalText define how unmarshal in TOML parsing
func (c *Constraint) UnmarshalText(text []byte) error {
	constraint, err := NewConstraint(string(text))
	if err != nil {
		return err
	}
	c.Key = constraint.Key
	c.MustMatch = constraint.MustMatch
	c.Regex = constraint.Regex
	return nil
}

var _ encoding.TextMarshaler = (*Constraint)(nil)

// MarshalText encodes the receiver into UTF-8-encoded text and returns the result.
func (c *Constraint) MarshalText() (text []byte, err error) {
	return []byte(c.String()), nil
}

// MatchConstraintWithAtLeastOneTag tests a constraint for one single service
func (c *Constraint) MatchConstraintWithAtLeastOneTag(tags []string) bool {
	for _, tag := range tags {
		if glob.Glob(c.Regex, tag) {
			return true
		}
	}
	return false
}

//Set []*Constraint
func (cs *Constraints) Set(str string) error {
	exps := strings.Split(str, ",")
	if len(exps) == 0 {
		return errors.New("Bad Constraint format: " + str)
	}
	for _, exp := range exps {
		constraint, err := NewConstraint(exp)
		if err != nil {
			return err
		}
		*cs = append(*cs, constraint)
	}
	return nil
}

// Constraints holds a Constraint parser
type Constraints []*Constraint

//Get []*Constraint
func (cs *Constraints) Get() interface{} { return []*Constraint(*cs) }

//String returns []*Constraint in string
func (cs *Constraints) String() string { return fmt.Sprintf("%+v", *cs) }

//SetValue sets []*Constraint into the parser
func (cs *Constraints) SetValue(val interface{}) {
	*cs = Constraints(val.(Constraints))
}

// Type exports the Constraints type as a string
func (cs *Constraints) Type() string {
	return fmt.Sprint("constraint")
}

// Store holds KV store cluster config
type Store struct {
	store.Store
	Prefix string // like this "prefix" (without the /)
}

// Cluster holds cluster config
type Cluster struct {
	Node  string `description:"Node name"`
	Store *Store
}

// Auth holds authentication configuration (BASIC, DIGEST, users)
type Auth struct {
	Basic       *Basic
	Digest      *Digest
	HeaderField string
}

// Users authentication users
type Users []string

// Basic HTTP basic authentication
type Basic struct {
	Users     `mapstructure:","`
	UsersFile string
}

// Digest HTTP authentication
type Digest struct {
	Users     `mapstructure:","`
	UsersFile string
}

// CanonicalDomain returns a lower case domain with trim space
func CanonicalDomain(domain string) string {
	return strings.ToLower(strings.TrimSpace(domain))
}

// Statistics provides options for monitoring request and response stats
type Statistics struct {
	RecentErrors int `description:"Number of recent errors logged"`
}

// Metrics provides options to expose and send Traefik metrics to different third party monitoring systems
type Metrics struct {
	Prometheus *Prometheus `description:"Prometheus metrics exporter type"`
}

// Prometheus can contain specific configuration used by the Prometheus Metrics exporter
type Prometheus struct {
	Buckets Buckets `description:"Buckets for latency metrics"`
}

// Buckets holds Prometheus Buckets
type Buckets []float64

//Set adds strings elem into the the parser
//it splits str on "," and ";" and apply ParseFloat to string
func (b *Buckets) Set(str string) error {
	fargs := func(c rune) bool {
		return c == ',' || c == ';'
	}
	// get function
	slice := strings.FieldsFunc(str, fargs)
	for _, bucket := range slice {
		bu, err := strconv.ParseFloat(bucket, 64)
		if err != nil {
			return err
		}
		*b = append(*b, bu)
	}
	return nil
}

//Get []float64
func (b *Buckets) Get() interface{} { return Buckets(*b) }

//String return slice in a string
func (b *Buckets) String() string { return fmt.Sprintf("%v", *b) }

//SetValue sets []float64 into the parser
func (b *Buckets) SetValue(val interface{}) {
	*b = Buckets(val.(Buckets))
}

// AccessLog holds the configuration settings for the access logger (middlewares/accesslog).
type AccessLog struct {
	FilePath string `json:"file,omitempty" description:"Access log file path. Stdout is used when omitted or empty"`
	Format   string `json:"format,omitempty" description:"Access log format: json | common"`
}

// Exclusion excludes a request from auditing if the http header contains any of the specified values
type Exclusion struct {
	HeaderName string   `json:"headerName,omitempty" description:"Request header name to evaluate"`
	Contains   []string `json:"contains,omitempty" description:"Substring values to exclude"`
	EndsWith   []string `json:"endsWith,omitempty" description:"End of string values to exclude"`
	StartsWith []string `json:"startsWith,omitempty" description:"Start of string values to exclude"`
}

// Enabled states whether any exclusion filters are specified
func (e *Exclusion) Enabled() bool {
	return len(e.Contains) > 0 || len(e.EndsWith) > 0 || len(e.StartsWith) > 0
}

// Exclusions is a container type for Exclusion
type Exclusions map[string]*Exclusion

// AuditSink holds AuditSink configuration
type AuditSink struct {
	Exclusions      Exclusions `json:"exclusions,omitempty"`
	Type            string     `json:"type,omitempty" description:"The type of sink: File/HTTP/Kafka/AMQP"`
	Endpoint        string     `json:"endpoint,omitempty" description:"Endpoint for audit tap. e.g. url for HTTP/Kafka/AMQP or filename for File"`
	Destination     string     `json:"destination,omitempty" description:"For Kafka the topic, AMQP the exchange etc."`
	MaxEntityLength string     `json:"maxEntityLength,omitempty" description:"MaxEntityLength truncates entities (bodies) longer than this (units are allowed, eg. 32KiB)"`
	NumProducers    int        `json:"numProducers,omitempty" description:"The number of concurrent producers which can send messages to the endpoint"`
	ChannelLength   int        `json:"channelLength,omitempty" description:"Size of the in-memory message channel.  Used as a buffer in case of Producer failure"`
	DiskStorePath   string     `json:"diskStorePath,omitempty" description:"Directory path for disk-backed persistent audit message queue"`
	ProxyingFor     string     `json:"proxyingFor,omitempty" description:"Defines the style of auditing event required. e.g API, RATE"`
	AuditSource     string     `json:"auditSource,omitempty" description:"Value to use for auditSource in audit message"`
	AuditType       string     `json:"auditType,omitempty" description:"Value to use for auditType in audit message"`
	EncryptSecret   string     `json:"encryptSecret,omitempty" description:"Key for encrypting failed events. If present events will be AES encrypted"`
}
