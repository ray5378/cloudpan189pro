package setting

import (
	"github.com/tickstep/cloudpan189-api/cloudpan"
	appsessionSvi "github.com/xxcheng123/cloudpan189-share/internal/services/appsession"
)

func buildSettingPanClient(session *appsessionSvi.Session) *cloudpan.PanClient {
	if session == nil {
		return nil
	}
	webToken := cloudpan.WebLoginToken{}
	if cookie := cloudpan.RefreshCookieToken(session.Token.SessionKey); cookie != "" {
		webToken.CookieLoginUser = cookie
	}
	return cloudpan.NewPanClient(webToken, session.Token)
}
