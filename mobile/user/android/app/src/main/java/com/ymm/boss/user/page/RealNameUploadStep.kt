package com.ymm.boss.user.page

import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.graphics.Matrix
import android.net.Uri
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.ErrorOutline
import androidx.compose.material.icons.outlined.PhotoCamera
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.AccountApi
import com.ymm.boss.user.api.Api
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.io.ByteArrayOutputStream
import kotlin.math.max

// 单面上传状态:待上传(虚线+相机) / 上传中(进度) / 已上传(预览+绿对勾) / 失败(红标,点击重试)。
private enum class UploadState { IDLE, UPLOADING, DONE, FAILED }

// 长边压到该值以下:JPEG q=80 输出,身份证照片通常 < 500KB,远低于任何中间件限制
// (公网反代 nginx 默认 client_max_body_size 1m,debug 包走的公网路径曾因此抛 413)。
private const val MAX_EDGE_PX = 1600
private const val JPEG_QUALITY = 80
private const val MAX_BYTES = 32_000_000

// Screen2 证件上传:拍摄要点 + 人像面/国徽面上传卡 + 提交认证(两面完成前禁用)。
@Composable
internal fun RNUploadStep(
    name: String, idNo: String, sms: String, frontId: Long, backId: Long,
    onFront: (Long) -> Unit, onBack_: (Long) -> Unit,
    onSubmitted: () -> Unit,
) {
    var submitting by remember { mutableStateOf(false) }
    var submitErr by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()

    RNSharedCard {
        Text("拍摄要点", fontSize = 14.sp, fontWeight = FontWeight.W600, color = RN.ink)
        RNFootnote("· 证件原件拍摄，四角完整、清晰无反光\n· 字迹、头像、有效期清晰可辨\n· 请勿翻拍复印件或屏幕照片")
    }
    Column(Modifier.fillMaxWidth().padding(top = 8.dp)) {
        CardHead("证件照片")
        Row(Modifier.fillMaxWidth().padding(horizontal = 16.dp)) {
            UploadCard("人像面", Modifier.weight(1f), onUploaded = onFront)
            Spacer(Modifier.size(12.dp))
            UploadCard("国徽面", Modifier.weight(1f), onUploaded = onBack_)
        }
    }
    if (submitErr.isNotBlank()) RNFootnote(submitErr, Color(0xFFFF2D2F))
    RNPrimaryButton("提交认证", enabled = frontId > 0 && backId > 0, loading = submitting) {
        submitting = true; submitErr = ""
        scope.launch {
            try {
                AccountApi.submitVerify("身份证", name, idNo, sms, frontId, backId)
                // 提交成功由父级重拉状态,按服务端结论(自动 PASS/FAIL/人工 PENDING)落对应状态页
                onSubmitted()
            } catch (e: Exception) {
                submitErr = com.ymm.boss.user.api.Api.friendlyMessage(e)
            } finally { submitting = false }
        }
    }
}

/** 上传卡:自持状态机;选图→压缩→上传→DONE 回调附件 id;失败可点击重选。 */
@Composable
private fun UploadCard(label: String, modifier: Modifier = Modifier, onUploaded: (Long) -> Unit) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var state by remember { mutableStateOf(UploadState.IDLE) }
    var preview by remember { mutableStateOf<Bitmap?>(null) }
    var hint by remember { mutableStateOf("") }

    val picker = rememberLauncherForActivityResult(ActivityResultContracts.PickVisualMedia()) { uri ->
        if (uri != null) startUpload(scope, context, uri) { st, bmp, id, msg ->
            state = st; if (bmp != null) preview = bmp
            hint = msg
            if (id > 0) onUploaded(id)
        }
    }
    Column(modifier, horizontalAlignment = Alignment.CenterHorizontally) {
        Box(
            Modifier.fillMaxWidth().height(110.dp)
                .let { m ->
                    if (state == UploadState.IDLE) m.border(1.dp, RN.placeholder, RoundedCornerShape(8.dp))
                    else m.background(RN.line.copy(alpha = 0.3f), RoundedCornerShape(8.dp))
                }
                .clickable {
                    picker.launch(PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly))
                },
            contentAlignment = Alignment.Center,
        ) {
            when (state) {
                UploadState.IDLE -> Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Icon(Icons.Outlined.PhotoCamera, contentDescription = null,
                        tint = RN.placeholder, modifier = Modifier.size(28.dp))
                    Spacer(Modifier.height(6.dp))
                    Text("上传证件$label", fontSize = 12.sp, color = RN.muted)
                }
                UploadState.UPLOADING -> CircularProgressIndicator(Modifier.size(24.dp), strokeWidth = 2.dp)
                else -> preview?.let {
                    Image(it.asImageBitmap(), contentDescription = null, contentScale = ContentScale.Crop,
                        modifier = Modifier.fillMaxWidth().height(110.dp))
                } ?: Text("已上传", fontSize = 12.sp, color = RN.muted)
            }
            if (state == UploadState.DONE) Icon(
                Icons.Filled.CheckCircle, contentDescription = null, tint = RN.success,
                modifier = Modifier.align(Alignment.TopEnd).padding(4.dp).size(20.dp))
            if (state == UploadState.FAILED) Icon(
                Icons.Filled.ErrorOutline, contentDescription = null, tint = Color(0xFFFF2D2F),
                modifier = Modifier.align(Alignment.TopEnd).padding(4.dp).size(20.dp))
        }
        Spacer(Modifier.height(6.dp))
        Text(
            when (state) {
                UploadState.DONE -> "$label · 已上传"
                UploadState.UPLOADING -> "$label · 上传中…"
                UploadState.FAILED -> "$label · 上传失败，点击重试"
                else -> "$label · 待上传"
            },
            fontSize = 12.sp, color = if (state == UploadState.DONE) RN.success else RN.muted)
        if (hint.isNotBlank() && state == UploadState.FAILED) Text(hint, fontSize = 11.sp, color = Color(0xFFFF2D2F))
    }
}

/** 选图→读字节→压缩→上传;IOException 等中间件层错误(nginx 413)走 Api.friendlyMessage 统一映射。 */
private fun startUpload(
    scope: kotlinx.coroutines.CoroutineScope,
    context: Context, uri: Uri,
    onResult: (UploadState, Bitmap?, Long, String) -> Unit,
) {
    scope.launch {
        onResult(UploadState.UPLOADING, decodePreview(context, uri), 0, "")
        try {
            val id = withContext(Dispatchers.IO) {
                val compressed = compressForUpload(context, uri)
                    ?: throw IllegalStateException("图片处理失败")
                if (compressed.size > MAX_BYTES) throw IllegalStateException("图片超过 32MB 上限")
                AccountApi.uploadAttachment("idcard.jpg", "image/jpeg", compressed).optLong("id", 0L)
            }
            if (id > 0) onResult(UploadState.DONE, null, id, "")
            else throw IllegalStateException("上传响应缺少附件 id")
        } catch (e: Exception) {
            // 上传是 multipart:本仓库 Api.HTTP 层只翻 2xx 外的 status,413/网络错走 friendlyMessage;
            // 这里再加一层 "过大" 兜底文案,直白告诉用户怎么修。
            val friendly = Api.friendlyMessage(e)
            val msg = when {
                e is Api.HttpError && e.status == 413 -> "图片过大，请重新拍照或选择（建议 ≤2MB）"
                e is java.io.IOException && friendly.contains("网络", ignoreCase = true) ->
                    "网络异常，请检查 Wi-Fi 后重试"
                else -> friendly
            }
            onResult(UploadState.FAILED, null, 0, msg)
        }
    }
}

/** 读取 EXIF 朝向后解码,长边限制到 MAX_EDGE_PX,JPEG q=80 输出。 */
private fun compressForUpload(context: Context, uri: Uri): ByteArray? {
    val resolver = context.contentResolver
    val bounds = BitmapFactory.Options().apply { inJustDecodeBounds = true }
    resolver.openInputStream(uri)?.use { BitmapFactory.decodeStream(it, null, bounds) }
        ?: return null
    val (w, h) = bounds.outWidth to bounds.outHeight
    if (w <= 0 || h <= 0) return null
    val sample = computeInSampleSize(w, h, MAX_EDGE_PX)
    val opts = BitmapFactory.Options().apply { inSampleSize = sample }
    val raw = resolver.openInputStream(uri)?.use { BitmapFactory.decodeStream(it, null, opts) } ?: return null
    val rotated = applyExifOrientation(resolver, uri, raw)
    val scaled = scaleLongEdge(rotated, MAX_EDGE_PX)
    return ByteArrayOutputStream().use { out ->
        scaled.compress(Bitmap.CompressFormat.JPEG, JPEG_QUALITY, out)
        out.toByteArray()
    }
}

/** inSampleSize 必须是 2 的幂;按长边算出最大样本,既能降内存又能保清晰。 */
internal fun computeInSampleSize(w: Int, h: Int, maxEdge: Int): Int {
    var sample = 1
    val longEdge = max(w, h)
    while (longEdge / sample > maxEdge * 2) sample *= 2
    return sample
}

/** 等比缩放到长边 ≤ maxEdge;原图已 ≤ maxEdge 时直接返回。 */
private fun scaleLongEdge(src: Bitmap, maxEdge: Int): Bitmap {
    val longEdge = max(src.width, src.height)
    if (longEdge <= maxEdge) return src
    val ratio = maxEdge.toFloat() / longEdge
    val nw = (src.width * ratio).toInt().coerceAtLeast(1)
    val nh = (src.height * ratio).toInt().coerceAtLeast(1)
    return Bitmap.createScaledBitmap(src, nw, nh, true)
}

/** 读取 EXIF orientation 并旋转;部分相机竖拍未旋转时直接压出会侧躺。 */
private fun applyExifOrientation(
    resolver: android.content.ContentResolver, uri: Uri, bitmap: Bitmap,
): Bitmap {
    val deg = runCatching {
        resolver.openInputStream(uri)?.use { input ->
            val exif = androidx.exifinterface.media.ExifInterface(input)
            when (exif.getAttributeInt(
                androidx.exifinterface.media.ExifInterface.TAG_ORIENTATION,
                androidx.exifinterface.media.ExifInterface.ORIENTATION_NORMAL,
            )) {
                androidx.exifinterface.media.ExifInterface.ORIENTATION_ROTATE_90 -> 90f
                androidx.exifinterface.media.ExifInterface.ORIENTATION_ROTATE_180 -> 180f
                androidx.exifinterface.media.ExifInterface.ORIENTATION_ROTATE_270 -> 270f
                else -> 0f
            }
        } ?: 0f
    }.getOrDefault(0f)
    if (deg == 0f) return bitmap
    val matrix = Matrix().apply { postRotate(deg) }
    return Bitmap.createBitmap(bitmap, 0, 0, bitmap.width, bitmap.height, matrix, true)
}

/** UI 预览缩略图,大尺寸仍用 inSampleSize=4 降内存。 */
private fun decodePreview(context: Context, uri: Uri): Bitmap? = runCatching {
    val opts = BitmapFactory.Options().apply { inSampleSize = 4 }
    context.contentResolver.openInputStream(uri)?.use { BitmapFactory.decodeStream(it, null, opts) }
}.getOrNull()
