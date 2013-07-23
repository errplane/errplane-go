# Usage

```GO
import (
  "time"
  "github.com/errplane/errplane-go"
)

client := errplane.New("myapp", "production", "my key")

client.Report("some.metric", 123.4, time.Now(), "context string, e.g. exception message", errplane.Dimensions {
  "server": "fqdn.hostname"
})
```
