package io.gitpod.toolbox.service

import com.jetbrains.toolbox.gateway.PluginSettingsStore
import com.jetbrains.toolbox.gateway.ToolboxServiceLocator
import com.jetbrains.toolbox.gateway.connection.ClientHandle
import com.jetbrains.toolbox.gateway.connection.ClientHelper
import com.jetbrains.toolbox.gateway.ssh.validation.SshConnectionValidator
import com.jetbrains.toolbox.gateway.ui.ToolboxUi
import io.gitpod.toolbox.auth.GitpodAuthManager
import kotlinx.coroutines.CoroutineScope
import okhttp3.OkHttpClient
import java.util.concurrent.atomic.AtomicBoolean

object Utils {
    lateinit var sharedServiceLocator: ToolboxServiceLocator private set
    lateinit var coroutineScope: CoroutineScope private set
    lateinit var settingStore: PluginSettingsStore private set
    lateinit var sshConnectionValidator: SshConnectionValidator private set
    lateinit var httpClient: OkHttpClient private set
    lateinit var clientHelper: ClientHelper private set

    lateinit var dataManager: DataManager private set

    lateinit var toolboxUi: ToolboxUi private set


    fun initialize(serviceLocator: ToolboxServiceLocator) {
        if (!isInitialized.compareAndSet(false, true)) {
            return
        }
        sharedServiceLocator = serviceLocator
        coroutineScope = serviceLocator.getService(CoroutineScope::class.java)
        toolboxUi = serviceLocator.getService(ToolboxUi::class.java)
        settingStore = serviceLocator.getService(PluginSettingsStore::class.java)
        sshConnectionValidator = serviceLocator.getService(SshConnectionValidator::class.java)
        httpClient = serviceLocator.getService(OkHttpClient::class.java)
        clientHelper = serviceLocator.getService(ClientHelper::class.java)
        dataManager = DataManager()
    }

    fun openUrl(url: String) {
        toolboxUi.openUrl(url)
    }

    private val isInitialized = AtomicBoolean(false)
}