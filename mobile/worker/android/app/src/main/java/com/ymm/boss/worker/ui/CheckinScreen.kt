package com.ymm.boss.worker.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.TicketApi
import com.ymm.boss.worker.util.LocationHelper
import kotlinx.coroutines.launch

@Composable
fun CheckinScreen(nav: NavHost, no: String) {
    val info by loadOnce(no) { TicketApi.detail(no) }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current
    val toastOk = stringResource(R.string.checkin_toast_ok)
    val locFail = stringResource(R.string.checkin_loc_fail)
    val locOkFmt = stringResource(R.string.checkin_loc_ok)
    var locating by remember { mutableStateOf(false) }
    var locationInfo by remember { mutableStateOf("") }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar(stringResource(R.string.checkin_title), onBack = { nav.pop() })
        when (val s = info) {
            is Load.Loading -> Loading()
            is Load.Fail -> Card(Modifier.padding(14.dp)) { Notice(stringResource(R.string.checkin_load_fail), red = true) }
            is Load.Ok -> {
                val d = s.data
                Card(Modifier.padding(12.dp)) {
                    KvRow(stringResource(R.string.td_ticket_no), d.optString("ticketNo"))
                    KvRow(stringResource(R.string.td_customer), "${d.optString("customerName")}(${d.optString("customerPhoneMasked")})")
                    KvRow(stringResource(R.string.scan_kv_address), d.optString("address"))
                    KvRow(stringResource(R.string.transfer_kv_slot), d.optString("scheduleSlot"))
                }
                Card(Modifier.padding(12.dp)) { Notice(stringResource(R.string.checkin_notice)) }
                if (locationInfo.isNotEmpty()) {
                    Card(Modifier.padding(12.dp)) { Text(locationInfo, fontSize = 13.sp) }
                }
                Card(Modifier.padding(12.dp)) {
                    PrimaryButton(
                        text = if (locating) stringResource(R.string.checkin_btn_locating) else stringResource(R.string.checkin_btn_submit),
                        modifier = Modifier.fillMaxWidth(),
                        enabled = !locating,
                    ) {
                        locating = true
                        scope.launch {
                            try {
                                val loc = LocationHelper.getCurrentLocation(ctx)
                                val latLng = if (loc != null) {
                                    locationInfo = ctx.getString(R.string.checkin_loc_ok,
                                        loc.lat.toString(), loc.lng.toString(), loc.accuracy.toString())
                                    loc.lat to loc.lng
                                } else {
                                    locationInfo = locFail
                                    null
                                }
                                if (latLng == null) {
                                    locating = false
                                    return@launch
                                }
                                val (lat, lng) = latLng
                                val r = TicketApi.checkin(no, lat, lng)
                                toast(ctx, r.optString("message", toastOk))
                                nav.pop()
                            } catch (e: Exception) { toast(ctx, ctx.getString(R.string.checkin_toast_fail, e.message ?: "")) }
                            locating = false
                        }
                    }
                }
            }
        }
        Spacer(Modifier.height(12.dp))
    }
}
