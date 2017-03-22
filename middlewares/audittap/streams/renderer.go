package streams

import (
	. "github.com/containous/traefik/middlewares/audittap/audittypes"
)

// Renderer is a function that encodes an audit summary.
type Renderer func(Summary) Encoded

//-------------------------------------------------------------------------------------------------

// DirectJSONRenderer is a Renderer that directly converts the summary to JSON.
func DirectJSONRenderer(summary Summary) Encoded {
	return summary.ToJson()
}
