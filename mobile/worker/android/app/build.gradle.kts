import java.util.Properties

plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.android)
    alias(libs.plugins.kotlin.compose)
}

fun bossBaseUrl(default: String): String =
    (project.findProperty("bossBaseUrl") as? String)?.takeIf { it.isNotBlank() } ?: default

// 极光 AppKey:-PjpushAppKey=... 注入;凭据未到位时空串(SDK 打日志不 crash),到位后 CI 传参。
fun jpushAppKey(): String = (project.findProperty("jpushAppKey") as? String)?.takeIf { it.isNotBlank() } ?: ""

val keystoreProperties = Properties().apply {
    val f = rootProject.file("keystore.properties")
    if (f.exists()) f.inputStream().use { load(it) }
}
val hasReleaseKeystore = keystoreProperties.getProperty("storeFile") != null

android {
    namespace = "com.ymm.boss.worker"
    compileSdk = 36

    defaultConfig {
        applicationId = "com.ymm.boss.worker"
        minSdk = 26
        targetSdk = 36
        versionCode = 1
        versionName = "0.1.0"
        // 防 user 端同款坑:缺省老 InstrumentationTestRunner 对 JUnit4 静默跑 0 用例(目前无 androidTest,先占位)
        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
        buildConfigField("String", "JPUSH_APPKEY", "\"${jpushAppKey()}\"")
        // JPush AAR manifest 自带 ${JPUSH_APPKEY}/${JPUSH_CHANNEL} 占位,经 placeholders 注入。
        manifestPlaceholders["JPUSH_APPKEY"] = jpushAppKey()
        manifestPlaceholders["JPUSH_CHANNEL"] = "developer-default"
    }

    signingConfigs {
        create("release") {
            if (hasReleaseKeystore) {
                storeFile = rootProject.file(keystoreProperties.getProperty("storeFile"))
                storePassword = keystoreProperties.getProperty("storePassword")
                keyAlias = keystoreProperties.getProperty("keyAlias")
                keyPassword = keystoreProperties.getProperty("keyPassword")
            }
        }
    }

    buildTypes {
        debug {
            buildConfigField("String", "BOSS_BASE_URL", "\"${bossBaseUrl("http://43.240.223.138:28080/api/worker/v1")}\"")
        }
        release {
            isMinifyEnabled = true
            buildConfigField("String", "BOSS_BASE_URL", "\"${bossBaseUrl("https://boss.ymm.cn/api/worker/v1")}\"")
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro"
            )
            signingConfig = if (hasReleaseKeystore) signingConfigs.getByName("release")
            else signingConfigs.getByName("debug")
        }
    }
    buildFeatures {
        buildConfig = true
    }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
    kotlinOptions {
        jvmTarget = "17"
    }
}

dependencies {
    implementation(libs.androidx.core.ktx)
    implementation(libs.androidx.lifecycle.runtime.ktx)
    implementation(libs.androidx.activity.compose)
    implementation(platform(libs.androidx.compose.bom))
    implementation(libs.androidx.compose.ui)
    implementation(libs.androidx.compose.ui.graphics)
    implementation(libs.androidx.compose.ui.tooling.preview)
    implementation(libs.androidx.compose.material3)
    implementation(libs.androidx.compose.material.icons)
    implementation("androidx.compose.material:material-icons-extended")
    implementation(libs.kotlinx.coroutines.android)
    debugImplementation(libs.androidx.compose.ui.tooling)
    testImplementation(libs.junit)
    testImplementation("org.json:json:20231013") // 本地单测用真 org.json(android.jar 里是抛异常的 stub,与 user 端一致)
    androidTestImplementation(libs.androidx.junit)
    androidTestImplementation(libs.androidx.espresso.core)
    androidTestImplementation(platform(libs.androidx.compose.bom))
    androidTestImplementation(libs.androidx.compose.ui.test.junit4)
    androidTestImplementation(libs.androidx.compose.ui.test.manifest)

    // CameraX 相机
    implementation(libs.androidx.camerax.core)
    implementation(libs.androidx.camerax.camera2)
    implementation(libs.androidx.camerax.lifecycle)
    implementation(libs.androidx.camerax.view)

    // ML Kit 条码扫描
    implementation(libs.mlkit.barcode.scanning)

    // 定位签到
    implementation(libs.play.services.location)

    // Guava (CameraX ListenableFuture)
    implementation(libs.guava.android)

    // 极光推送(MavenCentral;jcore 经传递依赖引入)
    implementation("cn.jiguang.sdk:jpush:5.7.0")
}
