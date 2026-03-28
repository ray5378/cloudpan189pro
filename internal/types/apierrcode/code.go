package apierrcode

import (
	"net/http"
	"strings"
)

// Code 错误码类型
type Code string

// Desc 错误描述类型
type Desc string

// HTTPCode HTTP状态码类型
type HTTPCode int

// Error 错误定义结构
type Error struct {
	Code     Code
	Desc     Desc
	HTTPCode HTTPCode
}

func (e Error) Error() string {
	return string(e.Desc)
}

var list = make([]Code, 0)

func define(content string) (code Code) {
	defer func() {
		list = append(list, code)
	}()

	return Code(content)
}

// GetError 根据错误码获取错误信息
func getError(code Code) Error {
	return ErrorMap[code]
}

func As(err error) (error, bool) {
	if err == nil {
		return nil, false
	}

	for _, code := range list {
		// 匹配
		if strings.Contains(err.Error(), string(code)) {
			return getError(code), true
		}
	}

	return err, false
}

// 天翼云盘API错误码定义
var (
	// 访问控制相关错误
	AccessDenyOfHighFrequency Code = define("AccessDenyOfHighFrequency")

	// 下载相关错误
	ErrorDownloadFileNotFound          Code = define("ErrorDownloadFileNotFound")
	ErrorDownloadFileDeleted           Code = define("ErrorDownloadFileDeleted")
	ErrorDownloadFileInvalidParam      Code = define("ErrorDownloadFileInvalidParam")
	ErrorDownloadFileInternalError     Code = define("ErrorDownloadFileInternalError")
	ErrorDownloadFileInvalidSessionKey Code = define("ErrorDownloadFileInvalidSessionKey")
	ErrorDownloadFileShareTimeOut      Code = define("ErrorDownloadFileShareTimeOut")

	// 文件操作相关错误
	FileAlreadyExists        Code = define("FileAlreadyExists")
	FileNotFound             Code = define("FileNotFound")
	FileTooLarge             Code = define("FileTooLarge")
	InsufficientStorageSpace Code = define("InsufficientStorageSpace")
	InvalidParentFolder      Code = define("InvalidParentFolder")
	ParentNotFolder          Code = define("ParentNotFolder")
	MoveFileValidError       Code = define("MoveFileValidError")

	// 系统相关错误
	InternalError     Code = define("InternalError")
	InvalidArgument   Code = define("InvalidArgument")
	InvalidPassword   Code = define("InvalidPassword")
	InvalidSessionKey Code = define("InvalidSessionKey")
	InvalidSignature  Code = define("InvalidSignature")
	PermissionDenied  Code = define("PermissionDenied")
	ServiceNotOpen    Code = define("ServiceNotOpen")

	// 用户相关错误
	NoSuchUser                Code = define("NoSuchUser")
	MyIDQRCodeNotLogin        Code = define("MyIDQRCodeNotLogin")
	MyIDSignatureVerfiyFailed Code = define("MyIDSignatureVerfiyFailed")
	QRCodeNotBind             Code = define("QRCodeNotBind")
	QRCodeNotFound            Code = define("QRCodeNotFound")
	NotFoundPersonQuestion    Code = define("NotFoundPersonQuestion")
	UserInvalidOpenToken      Code = define("UserInvalidOpenToken")
	NotOpenAccount            Code = define("NotOpenAccount")

	// 上传相关错误
	UploadFileAccessViolation   Code = define("UploadFileAccessViolation")
	UploadFileNotFound          Code = define("UploadFileNotFound")
	UploadFileSaveFailed        Code = define("UploadFileSaveFailed")
	UploadFileVerifyFailed      Code = define("UploadFileVerifyFailed")
	InvalidUploadFileStatus     Code = define("InvalidUploadFileStatus")
	UploadSingleFileOverLimited Code = define("UploadSingleFileOverLimited")

	// 分享相关错误
	ShareSpecialDirError        Code = define("ShareSpecialDirError")
	SpecialDirShareError        Code = define("SpecialDirShareError")
	ShareInfoNotFound           Code = define("ShareInfoNotFound")
	ShareOverLimitedNumber      Code = define("ShareOverLimitedNumber")
	ShareAuditNo                Code = define("ShareAuditNo")
	ShareAuditNotPass           Code = define("ShareAuditNotPass")
	ShareAuditWaiting           Code = define("ShareAuditWaiting")
	ShareDumpFileNumOverLimited Code = define("ShareDumpFileNumOverLimited")
	ShareNotFoundFlatDir        Code = define("ShareNotFoundFlatDir")
	ShareDumpFileOverload       Code = define("ShareDumpFileOverload")
	ShareNotFound               Code = define("ShareNotFound")
	ShareAccessOverload         Code = define("ShareAccessOverload")
	ShareCreateFailed           Code = define("ShareCreateFailed")
	ShareExpiredError           Code = define("ShareExpiredError")
	ShareFileNotBelong          Code = define("ShareFileNotBelong")
	ShareCreateOverload         Code = define("ShareCreateOverload")
	ShareNotReceiver            Code = define("ShareNotReceiver")
	ErrorAccessCode             Code = define("ErrorAccessCode")

	// 批量操作相关错误
	BatchOperSuccessed  Code = define("BatchOperSuccessed")
	BatchOperFileFailed Code = define("BatchOperFileFailed")

	// 信息安全相关错误
	InfoSecurityErrorCode Code = define("InfoSecurityErrorCode")
	InfosecuMD5CheckError Code = define("InfosecuMD5CheckError")
	TextAuditErrorCode    Code = define("TextAuditErrorCode")

	// 转存相关错误
	CopyFileOverLimitedSpaceError Code = define("CopyFileOverLimitedSpaceError")
	CopyFileOverLimitedNumError   Code = define("CopyFileOverLimitedNumError")
	FileCopyToSubFolderError      Code = define("FileCopyToSubFolderError")

	// 通用操作错误
	CommonOperNotSupport    Code = define("CommonOperNotSupport")
	CommonInvalidSessionKey Code = define("CommonInvalidSessionKey")
	PhotoNumOverLimited     Code = define("PhotoNumOverLimited")
	UserDayFlowOverLimited  Code = define("UserDayFlowOverLimited")
	AccountNoAccess         Code = define("AccountNoAccess")
	ErrorLogin              Code = define("ErrorLogin")
	InvalidAccessToken      Code = define("InvalidAccessToken")
	ObjectIdVerifyFailed    Code = define("ObjectIdVerifyFailed")
	ParasTimeOut            Code = define("ParasTimeOut")
	TokenAlreadyExist       Code = define("TokenAlreadyExist")
	PrivateFileError        Code = define("PrivateFileError")
	FileStatusInvalid       Code = define("FileStatusInvalid")

	// 支付相关错误
	PayMoneyNumErrorCode     Code = define("PayMoneyNumErrorCode")
	UserOrderNotExists       Code = define("UserOrderNotExists")
	CreateSaleOrderErrorCode Code = define("CreateSaleOrderErrorCode")

	// 智能提速相关错误
	SpeedOrderRecordExist    Code = define("SpeedOrderRecordExist")
	SpeedOrderRecordNotExist Code = define("SpeedOrderRecordNotExist")
	SpeedProdAlreadyOrder    Code = define("SpeedProdAlreadyOrder")
	SpeedDialaccountNotFound Code = define("SpeedDialaccountNotFound")
	SpeedNotGdBroadbandUser  Code = define("SpeedNotGdBroadbandUser")
	SpeedUnOrder             Code = define("SpeedUnOrder")
	SpeedInfoNotExist        Code = define("SpeedInfoNotExist")
	SpeedInfoAlreadyExist    Code = define("SpeedInfoAlreadyExist")
	UnSpeedUpError           Code = define("UnSpeedUpError")
)

// 错误码映射表
var ErrorMap = map[Code]Error{
	// 访问控制相关错误
	AccessDenyOfHighFrequency: {
		Code:     AccessDenyOfHighFrequency,
		Desc:     "由于访问频率过高，拒绝访问",
		HTTPCode: http.StatusBadRequest,
	},

	// 下载相关错误
	ErrorDownloadFileNotFound: {
		Code:     ErrorDownloadFileNotFound,
		Desc:     "下载时文件不存在",
		HTTPCode: http.StatusBadRequest,
	},
	ErrorDownloadFileDeleted: {
		Code:     ErrorDownloadFileDeleted,
		Desc:     "下载时文件已被删除",
		HTTPCode: http.StatusBadRequest,
	},
	ErrorDownloadFileInvalidParam: {
		Code:     ErrorDownloadFileInvalidParam,
		Desc:     "下载时无效的下载参数",
		HTTPCode: http.StatusBadRequest,
	},
	ErrorDownloadFileInternalError: {
		Code:     ErrorDownloadFileInternalError,
		Desc:     "下载时内部错误",
		HTTPCode: http.StatusBadRequest,
	},
	ErrorDownloadFileInvalidSessionKey: {
		Code:     ErrorDownloadFileInvalidSessionKey,
		Desc:     "下载时无效的sessionKey",
		HTTPCode: http.StatusBadRequest,
	},
	ErrorDownloadFileShareTimeOut: {
		Code:     ErrorDownloadFileShareTimeOut,
		Desc:     "下载时分享文件超时",
		HTTPCode: http.StatusBadRequest,
	},

	// 文件操作相关错误
	FileAlreadyExists: {
		Code:     FileAlreadyExists,
		Desc:     "文件或文件夹已存在",
		HTTPCode: http.StatusBadRequest,
	},
	FileNotFound: {
		Code:     FileNotFound,
		Desc:     "文件或文件夹不存在",
		HTTPCode: http.StatusBadRequest,
	},
	FileTooLarge: {
		Code:     FileTooLarge,
		Desc:     "上传文件超过最大限制",
		HTTPCode: http.StatusBadRequest,
	},
	InsufficientStorageSpace: {
		Code:     InsufficientStorageSpace,
		Desc:     "剩余存储空间不足",
		HTTPCode: http.StatusBadRequest,
	},
	InvalidParentFolder: {
		Code:     InvalidParentFolder,
		Desc:     "无效的父目录",
		HTTPCode: http.StatusBadRequest,
	},
	ParentNotFolder: {
		Code:     ParentNotFolder,
		Desc:     "父文件夹类型不正确",
		HTTPCode: http.StatusBadRequest,
	},
	MoveFileValidError: {
		Code:     MoveFileValidError,
		Desc:     "文件移动类型检查错误",
		HTTPCode: http.StatusBadRequest,
	},

	// 系统相关错误
	InternalError: {
		Code:     InternalError,
		Desc:     "内部错误",
		HTTPCode: http.StatusInternalServerError,
	},
	InvalidArgument: {
		Code:     InvalidArgument,
		Desc:     "非法参数",
		HTTPCode: http.StatusBadRequest,
	},
	InvalidPassword: {
		Code:     InvalidPassword,
		Desc:     "密码不正确",
		HTTPCode: http.StatusBadRequest,
	},
	InvalidSessionKey: {
		Code:     InvalidSessionKey,
		Desc:     "非法登录会话Key",
		HTTPCode: http.StatusBadRequest,
	},
	InvalidSignature: {
		Code:     InvalidSignature,
		Desc:     "非法签名",
		HTTPCode: http.StatusBadRequest,
	},
	PermissionDenied: {
		Code:     PermissionDenied,
		Desc:     "访问权限不足",
		HTTPCode: http.StatusBadRequest,
	},
	ServiceNotOpen: {
		Code:     ServiceNotOpen,
		Desc:     "云存储服务尚未开通",
		HTTPCode: http.StatusBadRequest,
	},

	// 用户相关错误
	NoSuchUser: {
		Code:     NoSuchUser,
		Desc:     "用户账号不存在",
		HTTPCode: http.StatusBadRequest,
	},
	MyIDQRCodeNotLogin: {
		Code:     MyIDQRCodeNotLogin,
		Desc:     "MyID二维码未登录",
		HTTPCode: http.StatusBadRequest,
	},
	MyIDSignatureVerfiyFailed: {
		Code:     MyIDSignatureVerfiyFailed,
		Desc:     "MyID数字签名验证失败",
		HTTPCode: http.StatusBadRequest,
	},
	QRCodeNotBind: {
		Code:     QRCodeNotBind,
		Desc:     "二维码未绑定",
		HTTPCode: http.StatusBadRequest,
	},
	QRCodeNotFound: {
		Code:     QRCodeNotFound,
		Desc:     "二维码不存在",
		HTTPCode: http.StatusBadRequest,
	},
	NotFoundPersonQuestion: {
		Code:     NotFoundPersonQuestion,
		Desc:     "没有设置个人问题",
		HTTPCode: http.StatusBadRequest,
	},
	UserInvalidOpenToken: {
		Code:     UserInvalidOpenToken,
		Desc:     "无效的天翼账号Token",
		HTTPCode: http.StatusBadRequest,
	},
	NotOpenAccount: {
		Code:     NotOpenAccount,
		Desc:     "手机号未创建天翼帐号",
		HTTPCode: http.StatusBadRequest,
	},

	// 上传相关错误
	UploadFileAccessViolation: {
		Code:     UploadFileAccessViolation,
		Desc:     "上传文件访问冲突",
		HTTPCode: http.StatusBadRequest,
	},
	UploadFileNotFound: {
		Code:     UploadFileNotFound,
		Desc:     "上传文件不存在",
		HTTPCode: http.StatusBadRequest,
	},
	UploadFileSaveFailed: {
		Code:     UploadFileSaveFailed,
		Desc:     "上传文件保存至云存储失败",
		HTTPCode: http.StatusBadRequest,
	},
	UploadFileVerifyFailed: {
		Code:     UploadFileVerifyFailed,
		Desc:     "上传文件校验失败",
		HTTPCode: http.StatusBadRequest,
	},
	InvalidUploadFileStatus: {
		Code:     InvalidUploadFileStatus,
		Desc:     "无效的上传文件状态",
		HTTPCode: http.StatusBadRequest,
	},
	UploadSingleFileOverLimited: {
		Code:     UploadSingleFileOverLimited,
		Desc:     "上传单文件大小超限",
		HTTPCode: http.StatusBadRequest,
	},

	// 分享相关错误
	ShareSpecialDirError: {
		Code:     ShareSpecialDirError,
		Desc:     "共享特殊目录",
		HTTPCode: http.StatusBadRequest,
	},
	SpecialDirShareError: {
		Code:     SpecialDirShareError,
		Desc:     "特殊目录分享",
		HTTPCode: http.StatusBadRequest,
	},
	ShareInfoNotFound: {
		Code:     ShareInfoNotFound,
		Desc:     "没有找到分享信息(可能被取消分享了)",
		HTTPCode: http.StatusBadRequest,
	},
	ShareOverLimitedNumber: {
		Code:     ShareOverLimitedNumber,
		Desc:     "分享次数超限",
		HTTPCode: http.StatusBadRequest,
	},
	ShareAuditNo: {
		Code:     ShareAuditNo,
		Desc:     "分享审核不通过",
		HTTPCode: http.StatusBadRequest,
	},
	ShareAuditNotPass: {
		Code:     ShareAuditNotPass,
		Desc:     "分享审核不通过",
		HTTPCode: http.StatusBadRequest,
	},
	ShareAuditWaiting: {
		Code:     ShareAuditWaiting,
		Desc:     "分享审核中",
		HTTPCode: http.StatusBadRequest,
	},
	ShareDumpFileNumOverLimited: {
		Code:     ShareDumpFileNumOverLimited,
		Desc:     "分享转存文件数超限",
		HTTPCode: http.StatusBadRequest,
	},
	ShareNotFoundFlatDir: {
		Code:     ShareNotFoundFlatDir,
		Desc:     "分享平铺目录未找到",
		HTTPCode: http.StatusBadRequest,
	},
	ShareDumpFileOverload: {
		Code:     ShareDumpFileOverload,
		Desc:     "分享转存文件数目超限",
		HTTPCode: http.StatusBadRequest,
	},
	ShareNotFound: {
		Code:     ShareNotFound,
		Desc:     "分享未找到",
		HTTPCode: http.StatusBadRequest,
	},
	ShareAccessOverload: {
		Code:     ShareAccessOverload,
		Desc:     "分享访问次数超限",
		HTTPCode: http.StatusBadRequest,
	},
	ShareCreateFailed: {
		Code:     ShareCreateFailed,
		Desc:     "分享创建失败",
		HTTPCode: http.StatusBadRequest,
	},
	ShareExpiredError: {
		Code:     ShareExpiredError,
		Desc:     "分享已过期",
		HTTPCode: http.StatusBadRequest,
	},
	ShareFileNotBelong: {
		Code:     ShareFileNotBelong,
		Desc:     "文件不属于当前分享文件或目录",
		HTTPCode: http.StatusBadRequest,
	},
	ShareCreateOverload: {
		Code:     ShareCreateOverload,
		Desc:     "用户创建分享次数超限",
		HTTPCode: http.StatusBadRequest,
	},
	ShareNotReceiver: {
		Code:     ShareNotReceiver,
		Desc:     "好友分享，访问者非接受者",
		HTTPCode: http.StatusBadRequest,
	},
	ErrorAccessCode: {
		Code:     ErrorAccessCode,
		Desc:     "分享访问码错误",
		HTTPCode: http.StatusBadRequest,
	},

	// 批量操作相关错误
	BatchOperSuccessed: {
		Code:     BatchOperSuccessed,
		Desc:     "批量操作部分成功",
		HTTPCode: http.StatusBadRequest,
	},
	BatchOperFileFailed: {
		Code:     BatchOperFileFailed,
		Desc:     "批量操作失败",
		HTTPCode: http.StatusBadRequest,
	},

	// 信息安全相关错误
	InfoSecurityErrorCode: {
		Code:     InfoSecurityErrorCode,
		Desc:     "违反信安规则",
		HTTPCode: http.StatusGone, // 410
	},
	InfosecuMD5CheckError: {
		Code:     InfosecuMD5CheckError,
		Desc:     "违反信安规则",
		HTTPCode: http.StatusBadRequest,
	},
	TextAuditErrorCode: {
		Code:     TextAuditErrorCode,
		Desc:     "敏感词检查不通过",
		HTTPCode: http.StatusBadRequest,
	},

	// 转存相关错误
	CopyFileOverLimitedSpaceError: {
		Code:     CopyFileOverLimitedSpaceError,
		Desc:     "转存文件总大小超限",
		HTTPCode: http.StatusBadRequest,
	},
	CopyFileOverLimitedNumError: {
		Code:     CopyFileOverLimitedNumError,
		Desc:     "转存次数超限",
		HTTPCode: http.StatusBadRequest,
	},
	FileCopyToSubFolderError: {
		Code:     FileCopyToSubFolderError,
		Desc:     "父目录拷贝或移动至自身子目录错误",
		HTTPCode: http.StatusBadRequest,
	},

	// 通用操作错误
	CommonOperNotSupport: {
		Code:     CommonOperNotSupport,
		Desc:     "操作不支持，建议升级版本",
		HTTPCode: http.StatusBadRequest,
	},
	CommonInvalidSessionKey: {
		Code:     CommonInvalidSessionKey,
		Desc:     "分享相关接口时，好友分享，需要登陆",
		HTTPCode: http.StatusBadRequest,
	},
	PhotoNumOverLimited: {
		Code:     PhotoNumOverLimited,
		Desc:     "照片数量超限",
		HTTPCode: http.StatusBadRequest,
	},
	UserDayFlowOverLimited: {
		Code:     UserDayFlowOverLimited,
		Desc:     "用户当日流量超过上限",
		HTTPCode: http.StatusBadRequest,
	},
	AccountNoAccess: {
		Code:     AccountNoAccess,
		Desc:     "出口ip不在白名单列表中",
		HTTPCode: http.StatusBadRequest,
	},
	ErrorLogin: {
		Code:     ErrorLogin,
		Desc:     "登录账号失败，paras只有1分钟有效，且只能请求一次",
		HTTPCode: http.StatusBadRequest,
	},
	InvalidAccessToken: {
		Code:     InvalidAccessToken,
		Desc:     "AccessToken无效",
		HTTPCode: http.StatusBadRequest,
	},
	ObjectIdVerifyFailed: {
		Code:     ObjectIdVerifyFailed,
		Desc:     "objectId校验失败",
		HTTPCode: http.StatusBadRequest,
	},
	ParasTimeOut: {
		Code:     ParasTimeOut,
		Desc:     "paras参数超时",
		HTTPCode: http.StatusBadRequest,
	},
	TokenAlreadyExist: {
		Code:     TokenAlreadyExist,
		Desc:     "该天翼云盘已绑定",
		HTTPCode: http.StatusBadRequest,
	},
	PrivateFileError: {
		Code:     PrivateFileError,
		Desc:     "异常操作私密空间文件错误",
		HTTPCode: http.StatusBadRequest,
	},
	FileStatusInvalid: {
		Code:     FileStatusInvalid,
		Desc:     "FMUserFile fileStatus is invalid",
		HTTPCode: http.StatusBadRequest,
	},

	// 支付相关错误
	PayMoneyNumErrorCode: {
		Code:     PayMoneyNumErrorCode,
		Desc:     "支付金额有误",
		HTTPCode: http.StatusBadRequest,
	},
	UserOrderNotExists: {
		Code:     UserOrderNotExists,
		Desc:     "用户订单不存在",
		HTTPCode: http.StatusBadRequest,
	},
	CreateSaleOrderErrorCode: {
		Code:     CreateSaleOrderErrorCode,
		Desc:     "创建用户订单异常",
		HTTPCode: http.StatusBadRequest,
	},

	// 智能提速相关错误
	SpeedOrderRecordExist: {
		Code:     SpeedOrderRecordExist,
		Desc:     "已经生成订购关系",
		HTTPCode: http.StatusBadRequest,
	},
	SpeedOrderRecordNotExist: {
		Code:     SpeedOrderRecordNotExist,
		Desc:     "没有找到订购关系",
		HTTPCode: http.StatusBadRequest,
	},
	SpeedProdAlreadyOrder: {
		Code:     SpeedProdAlreadyOrder,
		Desc:     "该宽带已订购该产品",
		HTTPCode: http.StatusBadRequest,
	},
	SpeedDialaccountNotFound: {
		Code:     SpeedDialaccountNotFound,
		Desc:     "找不到宽带账号",
		HTTPCode: http.StatusBadRequest,
	},
	SpeedNotGdBroadbandUser: {
		Code:     SpeedNotGdBroadbandUser,
		Desc:     "非广东宽带拨号用户",
		HTTPCode: http.StatusBadRequest,
	},
	SpeedUnOrder: {
		Code:     SpeedUnOrder,
		Desc:     "智能提速套餐未订购",
		HTTPCode: http.StatusBadRequest,
	},
	SpeedInfoNotExist: {
		Code:     SpeedInfoNotExist,
		Desc:     "智能提速套餐信息不存在",
		HTTPCode: http.StatusBadRequest,
	},
	SpeedInfoAlreadyExist: {
		Code:     SpeedInfoAlreadyExist,
		Desc:     "智能提速套餐信息已存在",
		HTTPCode: http.StatusBadRequest,
	},
	UnSpeedUpError: {
		Code:     UnSpeedUpError,
		Desc:     "用户处于未提速状态",
		HTTPCode: http.StatusBadRequest,
	},
}
