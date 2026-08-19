package com.ymm.boss.worker.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.ui.theme.Muted

// 未开发页面的占位实现,后续逐个替换为真实页面
@Composable
fun PlaceholderScreen(nav: NavHost, title: String, no: String? = null) {
    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar(title, onBack = { nav.pop() })
        Card(Modifier.padding(12.dp)) {
            Text("页面开发中", fontSize = 14.sp, color = Muted)
        }
    }
}