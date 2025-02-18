package common

import "fmt"

// GetDomainWithRegion returns the appropriate domain based on region
func GetDomainWithRegion(region string) string {
    if region != "" && region != "default" {
        return fmt.Sprintf("%s.portmap.io", region)
    }
    return "portmap.io"
}
