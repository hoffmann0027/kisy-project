package com.kisy.messenger.calls;

import android.content.Context;
import android.media.AudioDeviceCallback;
import android.media.AudioDeviceInfo;
import android.media.AudioManager;
import android.os.Build;
import android.os.Handler;
import android.os.Looper;
import android.os.PowerManager;
import android.util.Log;
import java.util.List;

/**
 * CallAudioSession's hands on a real phone: AudioManager for the route,
 * PowerManager for the proximity lock.
 *
 * Headsets are followed with an AudioDeviceCallback rather than the
 * ACTION_HEADSET_PLUG broadcast plus a BluetoothHeadset profile proxy. The
 * callback reports wired, USB and Bluetooth headsets alike, and — unlike the
 * profile proxy — needs no BLUETOOTH_CONNECT runtime permission on Android 12+,
 * which would have been one more system dialog before the first call.
 */
final class AndroidCallAudio implements CallAudioSession.Platform {

    private static final String TAG = "KisyCall";

    /**
     * How long after a device change the route is re-applied. The WebView's own
     * WebRTC audio code reacts to the same event and picks the loudspeaker when
     * a headset goes away; ours has to be the decision that lands last.
     */
    private static final long SETTLE_MS = 300;

    /**
     * A ceiling on the proximity lock, so that even a path nobody foresaw
     * cannot keep the screen dark for longer than any plausible call.
     */
    private static final long PROXIMITY_MAX_MS = 4 * 60 * 60 * 1000L;

    interface Listener {
        void onRoute(CallAudioSession.Route route, boolean speakerRequested);
    }

    private final AudioManager audio;
    private final PowerManager.WakeLock proximity;
    private final Handler main = new Handler(Looper.getMainLooper());
    private final Listener listener;
    private CallAudioSession session;
    private int savedMode = AudioManager.MODE_NORMAL;
    private boolean callbackRegistered;

    private final AudioDeviceCallback deviceCallback = new AudioDeviceCallback() {
        @Override
        public void onAudioDevicesAdded(AudioDeviceInfo[] added) {
            settle();
        }

        @Override
        public void onAudioDevicesRemoved(AudioDeviceInfo[] removed) {
            settle();
        }
    };

    AndroidCallAudio(Context context, Listener listener) {
        this.audio = (AudioManager) context.getSystemService(Context.AUDIO_SERVICE);
        PowerManager pm = (PowerManager) context.getSystemService(Context.POWER_SERVICE);
        // Tablets and some phones have no proximity sensor; there the call
        // simply keeps its screen on.
        this.proximity = pm != null && pm.isWakeLockLevelSupported(PowerManager.PROXIMITY_SCREEN_OFF_WAKE_LOCK)
            ? pm.newWakeLock(PowerManager.PROXIMITY_SCREEN_OFF_WAKE_LOCK, "kisy:call-proximity")
            : null;
        if (proximity != null) proximity.setReferenceCounted(false);
        this.listener = listener;
    }

    void attach(CallAudioSession session) {
        this.session = session;
    }

    private void settle() {
        main.removeCallbacksAndMessages(null);
        main.postDelayed(() -> {
            if (session != null) session.devicesChanged();
        }, SETTLE_MS);
    }

    @Override
    public void enterCallMode() {
        savedMode = audio.getMode();
        audio.setMode(AudioManager.MODE_IN_COMMUNICATION);
        if (!callbackRegistered) {
            audio.registerAudioDeviceCallback(deviceCallback, main);
            callbackRegistered = true;
        }
        Log.i(TAG, "call audio: communication mode");
    }

    @Override
    public void leaveCallMode() {
        main.removeCallbacksAndMessages(null);
        if (callbackRegistered) {
            audio.unregisterAudioDeviceCallback(deviceCallback);
            callbackRegistered = false;
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
            audio.clearCommunicationDevice();
        } else {
            audio.setSpeakerphoneOn(false);
            stopSco();
        }
        audio.setMode(savedMode == AudioManager.MODE_IN_COMMUNICATION ? AudioManager.MODE_NORMAL : savedMode);
        Log.i(TAG, "call audio: restored");
    }

    @Override
    public boolean hasWiredHeadset() {
        for (AudioDeviceInfo d : audio.getDevices(AudioManager.GET_DEVICES_OUTPUTS)) {
            if (isWired(d.getType())) return true;
        }
        return false;
    }

    @Override
    public boolean hasBluetoothHeadset() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
            for (AudioDeviceInfo d : audio.getAvailableCommunicationDevices()) {
                if (isBluetooth(d.getType())) return true;
            }
            return false;
        }
        for (AudioDeviceInfo d : audio.getDevices(AudioManager.GET_DEVICES_OUTPUTS)) {
            if (d.getType() == AudioDeviceInfo.TYPE_BLUETOOTH_SCO) return true;
        }
        return false;
    }

    @Override
    public void routeTo(CallAudioSession.Route route) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
            AudioDeviceInfo target = findCommunicationDevice(route);
            if (target != null) {
                boolean ok = audio.setCommunicationDevice(target);
                Log.i(TAG, "call audio: route " + route.wire + (ok ? "" : " REFUSED"));
            } else {
                Log.w(TAG, "call audio: no device for " + route.wire);
            }
            return;
        }
        // Before Android 12 there is no device to pick: the loudspeaker is a
        // switch, Bluetooth is an SCO link, and a wired headset takes the sound
        // by itself once the loudspeaker is off.
        audio.setSpeakerphoneOn(route == CallAudioSession.Route.SPEAKER);
        if (route == CallAudioSession.Route.BLUETOOTH) {
            audio.startBluetoothSco();
            audio.setBluetoothScoOn(true);
        } else {
            stopSco();
        }
        Log.i(TAG, "call audio: route " + route.wire);
    }

    @SuppressWarnings("deprecation")
    private void stopSco() {
        if (audio.isBluetoothScoOn()) {
            audio.setBluetoothScoOn(false);
            audio.stopBluetoothSco();
        }
    }

    private AudioDeviceInfo findCommunicationDevice(CallAudioSession.Route route) {
        List<AudioDeviceInfo> devices = audio.getAvailableCommunicationDevices();
        for (AudioDeviceInfo d : devices) {
            int t = d.getType();
            switch (route) {
                case EARPIECE:
                    if (t == AudioDeviceInfo.TYPE_BUILTIN_EARPIECE) return d;
                    break;
                case SPEAKER:
                    if (t == AudioDeviceInfo.TYPE_BUILTIN_SPEAKER) return d;
                    break;
                case WIRED:
                    if (isWired(t)) return d;
                    break;
                case BLUETOOTH:
                    if (isBluetooth(t)) return d;
                    break;
            }
        }
        return null;
    }

    private static boolean isWired(int type) {
        return type == AudioDeviceInfo.TYPE_WIRED_HEADSET
            || type == AudioDeviceInfo.TYPE_WIRED_HEADPHONES
            || type == AudioDeviceInfo.TYPE_USB_HEADSET;
    }

    private static boolean isBluetooth(int type) {
        if (type == AudioDeviceInfo.TYPE_BLUETOOTH_SCO) return true;
        return Build.VERSION.SDK_INT >= Build.VERSION_CODES.S && type == AudioDeviceInfo.TYPE_BLE_HEADSET;
    }

    @Override
    public void acquireProximityLock() {
        if (proximity == null) return;
        proximity.acquire(PROXIMITY_MAX_MS);
        Log.i(TAG, "call audio: proximity lock on");
    }

    @Override
    public void releaseProximityLock() {
        if (proximity == null || !proximity.isHeld()) return;
        // A plain release, not RELEASE_FLAG_WAIT_FOR_NO_PROXIMITY: when the
        // call is over the screen has to come back now, even if the phone is
        // still against the ear.
        proximity.release();
        Log.i(TAG, "call audio: proximity lock off");
    }

    @Override
    public boolean isProximityLockHeld() {
        return proximity != null && proximity.isHeld();
    }

    @Override
    public void publish(CallAudioSession.Route route, boolean speakerRequested) {
        listener.onRoute(route, speakerRequested);
    }
}
