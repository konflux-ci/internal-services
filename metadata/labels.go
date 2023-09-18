package metadata

import "fmt"

const (
	// rhtapDomain is the prefix of the application label
	rhtapDomain = "appstudio.openshift.io"
)

var (
	// pipelinesLabelPrefix is the prefix of the pipelines label
	pipelinesLabelPrefix = fmt.Sprintf("pipelines.%s", rhtapDomain)
)

var (
	// PipelinesTypeLabel is the label used to describe the type of pipeline
	PipelinesTypeLabel = fmt.Sprintf("%s/%s", pipelinesLabelPrefix, "type")
)
