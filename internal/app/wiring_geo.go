package app

// 地理 / GIS / ODN 域装配:三个域都是 PGStore 直绑,字段在 Application struct 内连续,
// 独立函数让 wiring.go 字面量不必关心内部实现细节。

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ymm-001/boss/internal/domain/geo"
	"github.com/ymm-001/boss/internal/domain/gis"
	"github.com/ymm-001/boss/internal/domain/odn"
)

func wireGeoServices(app *Application, pool *pgxpool.Pool) {
	app.Geo = geo.NewPGStore(pool)
	app.Gis = gis.NewPGStore(pool)
	app.ODN = odn.NewPGStore(pool)
}
