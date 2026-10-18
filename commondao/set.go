package commondao

import "context"

func SetWithCache[dataModel any, dbContextModel any, operateWithDBAndCache OperateWithDBAndCache[dataModel, dbContextModel]](ctx context.Context, DBContext DBContextInterface[dbContextModel], info operateWithDBAndCache) (err error) {
	err = info.SetDB(ctx, DBContext)
	if err != nil {
		return err
	}
	err = info.SetCache(ctx, DBContext)
	if err != nil {
		return err
	}
	return nil
} //事务在上层在执行回滚

func SetWithoutCache[dataModel any, dbContextModel any, operateWithDB OperateWithDB[dataModel, dbContextModel]](ctx context.Context, DBContext DBContextInterface[dbContextModel], info operateWithDB) error {
	return info.SetDB(ctx, DBContext)
}
