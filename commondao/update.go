package commondao

import "context"

func UpdateWithCache[dataModel any, dbContextModel any, operateWithDBAndCache OperateWithDBAndCache[dataModel, dbContextModel]](ctx context.Context, DBContext DBContextInterface[dbContextModel], info operateWithDBAndCache) (bool, error) {
	exist, err2 := info.UpdateDB(ctx, DBContext)
	err1 := info.DeleteCache(ctx, DBContext)
	if err2 != nil {
		return false, err2
	}
	if err1 != nil {
		return false, err1
	}
	return exist, nil
}
func UpdateWithoutCache[dataModel any, dbContextModel any, operateWithDB OperateWithDB[dataModel, dbContextModel]](ctx context.Context, DBContext DBContextInterface[dbContextModel], info operateWithDB) (bool, error) {
	return info.UpdateDB(ctx, DBContext)
}
