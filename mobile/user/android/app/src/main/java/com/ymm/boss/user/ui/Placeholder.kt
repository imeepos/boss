package com.ymm.boss.user.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

/** 未开发页面的统一占位,开发完成后替换。 */
@Composable
fun Placeholder(draft: String) {
    Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            Text("页面开发中", fontSize = 16.sp, color = Palette.ink)
            Text("草稿: docs/user/$draft", fontSize = 12.sp, color = Palette.muted, modifier = Modifier.padding(top = 4.dp))
        }
    }
}
