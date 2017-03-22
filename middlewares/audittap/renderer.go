package audittap

import (
	"encoding/json"
)

// Renderer is a function that encodes an audit summary.
type Renderer func(Summary) Encoded

//-------------------------------------------------------------------------------------------------

func (summary Summary) ToJson() Encoded {
	b, err := json.Marshal(summary)
	return Encoded{b, err}
}

// DirectJSONRenderer is a Renderer that directly converts the summary to JSON.
func DirectJSONRenderer(summary Summary) Encoded {
	return summary.ToJson()
}
