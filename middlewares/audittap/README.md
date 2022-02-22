# Audittap Configuration

Audit taps can be enabled by including appropriate configuration in the toml
configuration consumed by Traefik.

An example follows

```
[auditSink]
  type = "AMQP"
  endpoint = "amqp://localhost:5672/"
  destination = "audit"
  numProducers = 1
  channelLength = 1
  diskStorePath = "/tmp/goque"
  proxyingFor = "API"
  auditSource = "localSource"
  auditType = "localType"
  forwardXrequestId = true
  encryptSecret = "RDFXVxTgrrT9IseypJrwDLzk/nTVeTjbjaUR3RVyv94="
  maxAuditLength = "2M"
  maxPayloadContentsLength = "99K"
  maskFields = ["field1","field2","field3"]
  maskValue = "***"
  requestIdLabel = "govuk-tax"
  [auditSink.inclusions]
    [auditSink.inclusions.inc1]
    headerName = "RequestHost"
    matches = [".*\\.public\\.mdtp"]
    [auditSink.inclusions.inc2]
    headerName = "RequestPath"
    contains = ["/auditme/"]  
  [auditSink.exclusions]
    [auditSink.exclusions.exc1]
    headerName = "RequestHost"
    startsWith = ["captain", "docktor"]
    [auditSink.exclusions.exc2]
    headerName = "RequestPath"
    contains = ["/ping/ping"]
  [auditSink.headerMappings]
    [auditSink.headerMappings.detail]
    field1 = "header-1"
    [auditSink.headerMappings.tags]
    field2 = "header-2"    
  [auditSink.requestBodyCaptures]
    [auditSink.requestBodyCaptures.c1]
    headerName = "RequestHost"
    matches = [".*\\.public\\.mdtp"]
    [auditSink.requestBodyCaptures.c2]
    headerName = "RequestHost"
    endsWith = ["-frontend"]      
```

requestBodyCaptures

The properties are as follow:

* type (mandatory): the type of sink audit events will be published to. Can be AMQP|Blackhole
* proxyingFor (mandatory): determines the auditing style. Values can be API or RATE or MDTP
* auditSource (mandatory for API): the auditSource value to be included in API audit events
* auditType (mandatory for API): the auditType value to be included in API audit events
* forwardXrequestId (optional): if true will maintain any X-Request-ID in the request, otherwise creates a new one
* encryptSecret (optional): base64 encoded AES-256 key, if provided logged audit events will be encrypted
* maxAuditLength (optional): maximum byte length of audit defaulted to 100K. e.g 33K or 3M
* maxPayloadContentsLength (optional): maximum combined byte length of audit.requestPayload.contents and audit.responsePayload.contents. e.g 15K or 2M
* maskFields (optional): payload fields whose values should be replaced with maskValue if provided or the default
* maskValue (optional): the value to be used when masking is applied, default is *#########*
* requestIdLabel (optional): a value to be included as part of the X-Request-ID header. It will appear after any 's' prefix and before the UUID part
* auditSink.inclusions.incname (optional): include for auditing if header matches condition
* auditSink.exclusions.excname (optional): exclude for auditing if header matches condition
* headerMappings: dynamic extraction of headers into a section of the audit event
* requestBodyCaptures: include request body in the audit

### Notes
maskFields / maskValue behaviour varies depending on the proxyingFor style. Currently this is only applied for proxyingFor=MDTP in which case the masking is only applied for request/response payloads whose content type is _application/x-www-form-urlencoded_.

headerMappings, requestBodyCaptures are currently only applied by the MDTP audit tap

inclusions/exclusions filters control request auditing based on the header name when the header satisfies any of the specified values. Matching condition can be
    * contains
    * endsWith
    * startsWith
    * matches (a regex pattern)

Inclusions are applied prior to exclusions. So if inclusions are specified then at least 1 condition must be satisfied by the request before any exclusions will be evaluated.
 