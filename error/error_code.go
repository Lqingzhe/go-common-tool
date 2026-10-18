package newerror

// ============================================
// 错误码体系
// 0: 成功
// 1xxx: 请求参数错误
// 11xx: 认证授权错误
// 12xx: 资源错误
// 13xx: 业务规则错误
// 14xx: 流量控制错误
// 2xxx: 服务器内部错误
// 21xx: 服务依赖错误
// ============================================

type ErrorStatue int

const (
	CodeSuccess ErrorStatue = 0
)

// ---------- 1xxx: 请求参数错误 ----------
const (
	CodeInvalidParam          ErrorStatue = iota + 1001 // 请求参数错误
	CodeInvalidJSON                                     // JSON格式错误
	CodeMissingParam                                    // 缺少必要参数
	CodeParamFormatError                                // 参数格式错误（邮箱、手机号等）
	CodeParamValueInvalid                               // 参数值无效（超出范围、枚举值不对）
	CodeRequestBodyTooBig                               // 请求体过大
	CodeMethodNotAllowed                                // 请求方法不允许
	CodeUnsupportedMedia                                // 不支持的媒体类型（非application/json）
	CodeConnectionInterrupted                           // 连接中断
)

// ---------- 11xx: 认证授权错误 ----------
const (
	CodeUnauthorized        ErrorStatue = iota + 1101 // 未登录或Token已失效
	CodeAccessTokenExpired                            // AccessToken已过期
	CodeAccessTokenInvalid                            // AccessToken无效（签名错误或被篡改）
	CodeRefreshTokenInvalid                           // RefreshToken无效（不存在）
	CodePermissionDenied                              // 权限不足
	CodeAccountDisabled                               // 账号已被禁用
	CodeAccountNotActive                              // 账号未激活（邮箱或手机号未验证）
	CodeSecondAuthRequired                            // 需要二次认证
)

// ---------- 12xx: 资源错误 ----------
const (
	CodeUserNotFound         ErrorStatue = iota + 1201 // 用户不存在
	CodePasswordWrong                                  // 密码错误
	CodeUsernameExists                                 // 用户名已存在
	CodeResourceDuplicate                              // 资源重复/主键/联合索引冲突(除用户）
	CodeResourceNotFound                               // 资源不存在
	CodeResourceDeleted                                // 资源已被删除
	CodeResourceForbidden                              // 无权限访问该资源
	CodeResourceLocked                                 // 资源已被锁定
	CodeResourceExpired                                // 资源已过期
	CodeResourceLimitReached                           // 资源数量已达上限
)

// ---------- 13xx: 业务规则错误 ----------
const (
	CodeDataConflict        ErrorStatue = iota + 1301 // 数据冲突（并发修改）
	CodeOperationTooFreq                              // 操作过于频繁（业务层面）
	CodeDuplicateOperation                            // 请勿重复操作（已点赞、已关注等）
	CodeDependencyNotFound                            // 依赖数据不存在（关联数据已删除）
	CodeStateNotAllowed                               // 当前状态不允许该操作
	CodeInsufficientBalance                           // 余额不足（积分、余额）
)

// ---------- 14xx: 流量控制错误 ----------
const (
	CodeRateLimitExceeded ErrorStatue = iota + 1401 // 请求过于频繁（触发限流）
	CodeServiceBusy                                 // 服务繁忙（熔断中）
	CodeServiceDegraded                             // 服务降级中
)

// ---------- 2xxx: 服务器内部错误 ----------
const (
	CodeDatabaseError     ErrorStatue = iota + 2001 // 数据库异常
	CodeCacheError                                  // 缓存服务异常
	CodeMessageQueueError                           // 消息服务异常
	CodeFileOperationFail                           // 文件操作失败
	CodeNetworkTimeout                              // 网络超时
	CodeThirdPartyError                             // 第三方服务异常
	CodeInternalError                               // 系统内部错误（panic、未知错误）
	CodeDataInconsistent                            // 数据不一致
	CodeDependencyError                             // 依赖服务异常（依赖的微服务不可用）
)

// ---------- 21xx: 服务依赖错误 ----------
const (
	CodeServiceUnavailable ErrorStatue = iota + 2101 // 服务暂时不可用（重启或维护中）
	CodeUpstreamTimeout                              // 上游服务响应超时
	CodeDownstreamError                              // 下游服务调用失败
)

// ---------- 3xxx: 连接与实时通信错误 ----------
const (
	// 握手阶段

	CodeWebSocketUpgradeFailed   ErrorStatue = iota + 3001 // WebSocket 握手失败（缺少 Upgrade 头、Origin 校验不通过）
	CodeWebSocketOriginDenied                              //
	CodeWebSocketSubProtocolFail                           // 子协议协商失败

	// 连接生命周期

	CodeWebSocketClosedByPeer   // 对端主动关闭连接
	CodeWebSocketClosedByServer // 服务端主动关闭连接
	CodeWebSocketReadError      // 读取消息失败
	CodeWebSocketWriteError     // 写入消息失败
	CodeWebSocketPingTimeout    // 心跳超时（Pong 未按时返回）
	CodeWebSocketInvalidFrame   // 收到非法或无法解析的帧
	CodeWebSocketMessageTooBig  // 单条消息超过大小限制

	// 连接管理

	CodeConnectionNotFound      // 连接不存在（已断开但仍在尝Origin 校验被拒试写入）
	CodeConnectionLimitExceeded // 连接数超过限制
	CodeConnectionAuthFailed    // 连接鉴权失败（Token 无效/过期）
	CodeConnectionDuplicate     // 重复连接（同一用户重复建立连接）
)
