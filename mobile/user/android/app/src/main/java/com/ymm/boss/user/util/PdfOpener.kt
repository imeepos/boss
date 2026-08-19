package com.ymm.boss.user.util

import android.content.Context
import android.content.Intent
import android.net.Uri
import androidx.core.content.FileProvider
import com.ymm.boss.user.api.Api
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.File

/** 认证下载 PDF 到 cacheDir 后经 FileProvider 拉起系统查看器;成功 true,失败 false 由调用方 toast。 */
object PdfOpener {

    suspend fun openFromApi(context: Context, apiPath: String, fileName: String): Boolean =
        withContext(Dispatchers.IO) {
            try {
                val target = File(context.cacheDir, fileName)
                target.writeBytes(Api.getBytes(apiPath))
                context.startActivity(viewIntent(context, target))
                true
            } catch (e: Exception) {
                false
            }
        }

    private fun viewIntent(context: Context, file: File): Intent {
        val uri: Uri = FileProvider.getUriForFile(context, "${context.packageName}.fileprovider", file)
        return Intent(Intent.ACTION_VIEW)
            .setDataAndType(uri, "application/pdf")
            .addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
    }
}
