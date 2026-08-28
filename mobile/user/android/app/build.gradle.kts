import java.util.Properties

plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.android)
    alias(libs.plugins.kotlin.compose)
}

// -PbossBaseUrl=... 可覆盖默认值;移动端联调统一连接 192.168.0.102。
fun bossBaseUrl(default: String): String =
    (project.findProperty("bossBaseUrl") as? String)?.takeIf { it.isNotBlank() } ?: default

// keystore.properties 存在则用于 release 签名,否则回落 debug 签名保证 CI 可出包;keystore 本身不入库。
// 发布门禁: scripts/user-android-release-sign.sh 用 apksigner 阻断"回落 debug 签名"出包。
val keystoreProperties = Properties().apply {
    val f = rootProject.file("keystore.properties")
    if (f.exists()) f.inputStream().use { load(it) }
}
val hasReleaseKeystore = keystoreProperties.getProperty("storeFile") != null

android {
    namespace = "com.ymm.boss.user"
    compileSdk = 36

    defaultConfig {
        applicationId = "com.ymm.boss.user"
        minSdk = 26
        targetSdk = 36
        // 版号递增表(2026-08-27 决策):历史内测 debug 均 versionCode=1,首发须 >1 才能覆盖老机,
        // 故首发定 versionCode=2 / versionName=0.1.0;此后每次发版单调 +1,见 docs/notes/adopted/。
        versionCode = 2
        versionName = "0.1.0"
        // JUnit4/Compose 用例必须走 AndroidJUnitRunner;AGP 缺省注册老 InstrumentationTestRunner,connected 会静默跑 0 个用例
        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
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
            buildConfigField("String", "BOSS_BASE_URL", "\"${bossBaseUrl("http://43.240.223.138:28080/api/user/v1")}\"")
        }
        release {
            isMinifyEnabled = false
            // https 硬门槛(2026-08-27 用户拍板:首发含公网用户):release 走 https,正式域名
            // 定稿后仅需替换此处 host(https://192.168.0.102 为当前反代验证地址)。
            buildConfigField("String", "BOSS_BASE_URL", "\"${bossBaseUrl("https://192.168.0.102/api/user/v1")}\"")
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
    implementation(libs.androidx.security.crypto)
    implementation(libs.androidx.exifinterface)
    implementation(libs.stripe.android)
    implementation(libs.play.location)
    implementation(libs.kotlinx.coroutines.play.services)
    debugImplementation(libs.androidx.compose.ui.tooling)
    debugImplementation(libs.androidx.compose.ui.test.manifest)
    testImplementation(libs.junit)
    testImplementation("org.json:json:20231013") // 本地单测用真 org.json(android.jar 里是抛异常的 stub)
    androidTestImplementation(libs.androidx.junit)
    androidTestImplementation(libs.androidx.espresso.core)
    androidTestImplementation(platform(libs.androidx.compose.bom))
    androidTestImplementation(libs.androidx.compose.ui.test.junit4)
}
