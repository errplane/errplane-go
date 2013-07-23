# Usage

```GO
import (
  "time"
  "github.com/errplane/errplane-go"
)

client := errplane.New("w.apiv3.errplane.com", "udp.apiv3.errplane.com:8126", "myapp", "production", "my key")

client.Report("some.metric", 123.4, time.Now(), "context string, e.g. exception message", errplane.Dimensions {
  "server": "fqdn.hostname"
})
```
