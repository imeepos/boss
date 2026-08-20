package com.ymm.boss.worker.util

import android.Manifest
import android.content.pm.PackageManager
import android.util.Size
import androidx.camera.core.CameraSelector
import androidx.camera.core.ImageAnalysis
import androidx.camera.core.ImageProxy
import androidx.camera.core.Preview
import androidx.camera.lifecycle.ProcessCameraProvider
import androidx.camera.view.PreviewView
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.viewinterop.AndroidView
import androidx.core.content.ContextCompat
import androidx.lifecycle.compose.LocalLifecycleOwner
import com.google.common.util.concurrent.ListenableFuture
import com.google.mlkit.vision.barcode.BarcodeScanner
import com.google.mlkit.vision.barcode.BarcodeScanning
import com.google.mlkit.vision.common.InputImage
import java.util.concurrent.Executors

fun interface ScanCallback { fun onScanned(data: String) }

@Composable
fun CameraScanner(
    modifier: Modifier = Modifier,
    onScanned: ScanCallback,
    onError: ((String) -> Unit)? = null,
) {
    val ctx = LocalContext.current
    val lifecycleOwner = LocalLifecycleOwner.current
    val analyzerExecutor = remember { Executors.newSingleThreadExecutor() }
    val scanner: BarcodeScanner = remember { BarcodeScanning.getClient() }
    val hasCamera = ContextCompat.checkSelfPermission(ctx, Manifest.permission.CAMERA) ==
            PackageManager.PERMISSION_GRANTED

    Box(modifier = modifier) {
        if (!hasCamera) { onError?.invoke("相机权限未授予"); return@Box }
        AndroidView(
            factory = { context ->
                PreviewView(context).apply { scaleType = PreviewView.ScaleType.FILL_CENTER }
                    .also { previewView ->
                        val future: ListenableFuture<ProcessCameraProvider> = ProcessCameraProvider.getInstance(context)
                        future.addListener({
                            val provider: ProcessCameraProvider = future.get()
                            val preview = Preview.Builder().build().also { it.surfaceProvider = previewView.surfaceProvider }
                            val analysis = ImageAnalysis.Builder().setTargetResolution(Size(1280, 720))
                                .setBackpressureStrategy(ImageAnalysis.STRATEGY_KEEP_ONLY_LATEST).build()
                            analysis.setAnalyzer(analyzerExecutor) { proxy -> processImage(scanner, proxy, onScanned) }
                            try {
                                provider.unbindAll()
                                provider.bindToLifecycle(lifecycleOwner, CameraSelector.DEFAULT_BACK_CAMERA, preview, analysis)
                            } catch (e: Exception) { onError?.invoke("相机启动失败: ${e.message}") }
                        }, ContextCompat.getMainExecutor(context))
                    }
            },
            modifier = Modifier.fillMaxSize(),
        )
    }
    DisposableEffect(Unit) { onDispose { analyzerExecutor.shutdown(); scanner.close() } }
}

@Suppress("UnsafeOptInUsageError")
private fun processImage(scanner: BarcodeScanner, proxy: ImageProxy, cb: ScanCallback) {
    val mediaImage = proxy.image ?: run { proxy.close(); return }
    val input = InputImage.fromMediaImage(mediaImage, proxy.imageInfo.rotationDegrees)
    scanner.process(input).addOnSuccessListener { barcodes ->
        for (b in barcodes) {
            val v = b.rawValue
            if (v != null) { cb.onScanned(v); break }
        }
    }.addOnCompleteListener { proxy.close() }
}
