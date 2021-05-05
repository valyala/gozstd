package gozstd

import (
	"bytes"
	"sync"
)

var compInBufPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, cstreamInBufSize))
	},
}

var compOutBufPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, cstreamOutBufSize))
	},
}

var decInBufPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, dstreamInBufSize))
	},
}

var decOutBufPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, dstreamOutBufSize))
	},
}
