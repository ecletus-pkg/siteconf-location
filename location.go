package siteconf_location

import (
	"crypto/sha1"
	"fmt"
	"time"

	admin_field_location "github.com/ecletus-pkg/admin-field-location"
	admin_field_now "github.com/ecletus-pkg/admin-field-now"
	"github.com/moisespsena-go/aorm"
	"github.com/op/go-logging"

	"github.com/ecletus/fragment"

	admin_plugin "github.com/ecletus-pkg/admin"
	site_settings "github.com/ecletus-pkg/siteconf"
	"github.com/ecletus/admin"
	"github.com/ecletus/core"
	"github.com/ecletus/db"
	"github.com/ecletus/plug"
	"github.com/moisespsena-go/i18n-modular/i18nmod"
	path_helpers "github.com/moisespsena-go/path-helpers"
	"github.com/moisespsena-go/tzdb"
)

var (
	pkg         = path_helpers.GetCalledDir()
	group       = i18nmod.PkgToGroup(pkg)
	FieldPrefix = fmt.Sprintf("%x", sha1.Sum([]byte(pkg)))
	log         = logging.MustGetLogger(pkg)
)

type LocationKeyType uint8

func (this LocationKeyType) String() string {
	return site_settings.PrivateConfName(this)
}

const LocationKey LocationKeyType = 0

type Plugin struct {
	plug.EventDispatcher
	db.DBNames
	admin_plugin.AdminNames

	SitesRegisterKey string
}

func (p *Plugin) RequireOptions() []string {
	return []string{p.SitesRegisterKey}
}

func (p *Plugin) OnRegister(options *plug.Options) {
	admin_plugin.Events(p).InitResources(func(e *admin_plugin.AdminEvent) {
		e.Admin.OnResourceValueAdded(&site_settings.SiteConfigMain{}, func(e *admin.ResourceEvent) {
			e.Resource.AddFragmentConfig(&SiteConfigLocation{}, &admin.FragmentConfig{
				Config: &admin.Config{
					Setup: func(res *admin.Resource) {
						admin_field_location.Setup(res, "Location")
						admin_field_now.Setup(&admin_field_now.Field{
							LocationFunc: func(recorde interface{}, context *core.Context) *time.Location {
								location := res.GetMeta("Location", false).Value(context, recorde).(tzdb.LocationCity)
								return location.Location()
							},
						}, res)
						res.INESAttrs([][]string{{"Location", "Now"}})
					},
				},
			})
		})
	})

	db.Events(p).DBOnMigrate(func(e *db.DBEvent) error {
		return e.AutoMigrate(&SiteConfigLocation{}).Error
	})
}

func (p *Plugin) Init(options *plug.Options) {
	register := options.GetInterface(p.SitesRegisterKey).(*core.SitesRegister)
	register.SiteConfigGetter.Append(core.NewSiteGetter(func(site *core.Site, key interface{}) (value interface{}, ok bool) {
		if key == LocationKey {
			var config SiteConfigLocation
			if err := site.GetSystemDB().DB.First(&config, "fragment_enabled").Error; err == nil || aorm.IsRecordNotFoundError(err) {
				return &config.Location, true
			} else {
				log.Errorf("load config for site %s failed: %v", site.Name(), err)
			}
		}
		return
	}))
}

func Get(site *core.Site) (loc tzdb.Location, ok bool) {
	if v, ok := site.Config().Get(LocationKey); ok {
		return v.(tzdb.Location), true
	}
	return
}

func GetOrSys(site *core.Site) (tzdb.Location) {
	if v, ok := site.Config().Get(LocationKey); ok {
		return v.(tzdb.Location)
	}
	return tzdb.Sys
}

func GetC(ctx *core.Context) (loc tzdb.Location) {
	if i := ctx.Value(LocationKey); i != nil {
		if i != nil {
			return i.(tzdb.Location)
		}
		return nil
	}
	loc, _ = Get(ctx.Site)
	ctx.SetValue(LocationKey, loc)
	return
}

func GetOrSysC(ctx *core.Context) (loc tzdb.Location) {
	if i := ctx.Value(LocationKey); i != nil {
			return i.(tzdb.Location)
	}
	loc, _ = Get(ctx.Site)
	if loc == nil {
		loc = tzdb.Sys
	}
	ctx.SetValue(LocationKey, loc)
	return
}

type SiteConfigLocation struct {
	fragment.SingletonFormFragmentModel
	Location tzdb.LocationCity
}
