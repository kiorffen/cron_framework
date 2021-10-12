package dict

import (
	_ "context"
	_ "fmt"
	"time"
)

type TestDict struct {
	dict        []map[string]string
	active      int
	loading     int
	loadingFlag bool
}

func (c *TestDict) AutoReload(env string) {
	c.dict = make([]map[string]string, 2)
	c.active = 0
	c.load(env)
	go func() {
		for range time.Tick(10 * time.Minute) {
			c.load(env)
		}
	}()
}

func (c *TestDict) load(env string) {
	//fmt.Println(time.Now().Format("2006-01-02 15:04:05"))
	if c.loadingFlag == true {
		return
	}

	c.loadingFlag = true

	c.loading = 1 - c.active
	c.dict[c.loading] = make(map[string]string)

	//sql := fmt.Sprintf("select strategy_id,strategy_priority from tbRecoRuleStrategyInfo where 1")
	var rows []map[string]string
	var err error
	//if env == "release" {
	//	rows, err = G_mc["gicp3"].QueryString(context.Background(), sql)
	//} else {
	//	rows, err = G_mc["gicp3_dev"].QueryString(context.Background(), sql)
	//}

	if err != nil {
		return
	}

	if len(rows) == 0 {
		c.loadingFlag = false
		return
	}

	for _, row := range rows {
		c.dict[c.loading][row["strategy_id"]] = row["strategy_priority"]
	}

	c.active = c.loading

	c.loadingFlag = false
}

func (c *TestDict) Seek(strategy_id string) string {
	return c.dict[c.active][strategy_id]
}
