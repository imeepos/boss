package com.ymm.boss.worker

import android.app.Application
import cn.jpush.android.api.JPushInterface

// Application:JPush 通道初始化(官方要求在 Application#onCreate,不能放 Activity)。
// appKey 为空(凭据未到位)时 SDK 仅打日志,不影响 App 其余功能。
class BossApplication : Application() {
    override fun onCreate() {
        super.onCreate()
        JPushInterface.setDebugMode(BuildConfig.DEBUG)
        JPushInterface.init(this)
    }
}
