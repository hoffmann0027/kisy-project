import { createContext, useContext, type ReactNode } from "react";
import "./call.css";
import { useCall, type CallPeer } from "./useCall";
import { IncomingCallCard } from "./IncomingCallCard";
import { OutgoingCall } from "./OutgoingCall";
import { OngoingCall } from "./OngoingCall";

interface CallContextValue {
  // startCall begins an outgoing 1:1 audio call to a direct-chat peer.
  startCall: (peer: CallPeer, chatId: string) => void;
  // busy is true while any call (in/out/active) is in progress.
  busy: boolean;
}

const CallContext = createContext<CallContextValue | null>(null);

// useCallControls exposes call actions to the rest of the app (e.g. the
// conversation header's call button).
export function useCallControls(): CallContextValue {
  const ctx = useContext(CallContext);
  if (!ctx) throw new Error("useCallControls must be used within CallProvider");
  return ctx;
}

// CallProvider owns the single call session and renders its overlays above the
// app. Mounted once inside the authenticated layout so calls survive route
// changes and reach every page.
export function CallProvider({ children }: { children: ReactNode }) {
  const { view, startCall, accept, reject, hangup, toggleMute } = useCall();
  const busy = view.phase !== "idle" && view.phase !== "ended";

  return (
    <CallContext.Provider value={{ startCall: (peer, chatId) => void startCall(peer, chatId), busy }}>
      {children}

      {view.phase === "incoming" && view.peer && (
        <IncomingCallCard peer={view.peer} onAccept={() => void accept()} onReject={reject} />
      )}
      {view.phase === "outgoing" && view.peer && <OutgoingCall peer={view.peer} onCancel={hangup} />}
      {(view.phase === "connecting" || view.phase === "active") && (
        <OngoingCall view={view} onHangup={hangup} onToggleMute={toggleMute} />
      )}
      {view.phase === "ended" && view.endedReason && <div className="call-ended">{view.endedReason}</div>}
    </CallContext.Provider>
  );
}
