package buildvar

// -ldflags '-X "github.com/proletariatgames/p4harmonize/internal/buildvar.Version=${{ github.event.release.tag_name }}"'
var Version string

// -ldflags '-X "github.com/proletariatgames/p4harmonize/internal/buildvar.BuildTime=${{ github.event.release.created_at }}"'
var BuildTime string

// -ldflags '-X "github.com/proletariatgames/p4harmonize/internal/buildvar.ReleaseURL=${{ github.event.release.html_url }}"'
var ReleaseURL string
