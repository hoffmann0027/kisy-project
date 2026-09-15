package com.kisy.messenger;

import android.os.Bundle;
import com.getcapacitor.BridgeActivity;
import com.kisy.messenger.calls.KisyCallPlugin;

public class MainActivity extends BridgeActivity {

    @Override
    public void onCreate(Bundle savedInstanceState) {
        // Registered before super: the bridge collects plugins as it starts,
        // and one registered afterwards is invisible to the web layer.
        registerPlugin(KisyCallPlugin.class);
        super.onCreate(savedInstanceState);
    }
}
