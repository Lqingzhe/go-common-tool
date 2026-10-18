package commondao

import "context"

func DeleteWithCache[dataModel any, dbContextModel any, operateWithDBAndCache OperateWithDBAndCache[dataModel, dbContextModel]](ctx context.Context, dbContext DBContextInterface[dbContextModel], info operateWithDBAndCache) error {
	err := info.DeleteDB(ctx, dbContext)
	if err != nil {
		return err
	}
	err = info.DeleteCache(ctx, dbContext)
	if err != nil {
		return err
	}
	return nil
}
func DeleteWithoutCache[dataModel any, dbContextModel any, operateWithDB OperateWithDB[dataModel, dbContextModel]](ctx context.Context, dbContext DBContextInterface[dbContextModel], info operateWithDB) error {
	return info.DeleteDB(ctx, dbContext)
}
