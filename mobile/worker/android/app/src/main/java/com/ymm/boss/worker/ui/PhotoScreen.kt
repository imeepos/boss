package com.ymm.boss.worker.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.setValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.ScanApi
import java.io.File
import org.json.JSONArray
import kotlinx.coroutines.launch

@Composable
fun PhotoScreen(nav: NavHost, no: String) {
    var refresh by remember { mutableStateOf(0) }
    val state by loadOnce(no, refresh) { ScanApi.photos(no) }
    val ctx = LocalContext.current
    val scope = rememberCoroutineScope()
    var uploading by remember { mutableStateOf(false) }
    val picker = rememberLauncherForActivityResult(ActivityResultContracts.GetContent()) { uri ->
        if (uri == null) return@rememberLauncherForActivityResult
        uploading = true
        scope.launch {
            try {
                val bytes = ctx.contentResolver.openInputStream(uri)?.use { it.readBytes() }
                if (bytes == null) throw IllegalStateException(ctx.getString(R.string.photo_err_read))
                ScanApi.uploadPhoto(no, "evidence.jpg", ctx.contentResolver.getType(uri) ?: "image/jpeg", bytes)
                refresh++
                toast(ctx, ctx.getString(R.string.photo_toast_ok))
            } catch (e: Exception) { toast(ctx, ctx.getString(R.string.photo_err_load, e.message ?: "")) }
            uploading = false
        }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar(stringResource(R.string.photo_title), onBack = { nav.pop() })
        when (val s = state) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice(stringResource(R.string.photo_err_load, s.message), red = true) }
            is Load.Ok -> {
                val items = s.data.optJSONArray("items") ?: JSONArray()
                Card(Modifier.padding(12.dp)) {
                    KvRow(stringResource(R.string.photo_scene_title), stringResource(R.string.photo_scene_value))
                    SectionTitle(stringResource(R.string.photo_uploaded), more = stringResource(R.string.photo_count, items.length()))
                    if (items.length() == 0) Empty(stringResource(R.string.photo_empty))
                    for (i in 0 until items.length()) {
                        val it0 = items.optJSONObject(i)
                        KvRow(it0.optString("fileName"), stringResource(if (it0.optBoolean("linked")) R.string.photo_linked else R.string.photo_unlinked))
                    }
                }
                Card(Modifier.padding(12.dp)) {
                    PrimaryButton(stringResource(R.string.photo_pick), enabled = !uploading, modifier = Modifier.fillMaxWidth()) {
                        picker.launch("image/*")
                    }
                    Notice(stringResource(R.string.photo_notice))
                }
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}
