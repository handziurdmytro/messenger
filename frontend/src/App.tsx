import { useEffect, useState } from "react";
import { logout } from "./api/auth";
import { currentSession } from "./api/client";
import { Auth } from "./components/Auth";
import { Workspace } from "./components/Workspace";

export default function App() {
  const [signedIn, setSignedIn] = useState(() => !!currentSession());

  useEffect(() => {
    const expired = () => setSignedIn(false);
    window.addEventListener("session-expired", expired);
    return () => window.removeEventListener("session-expired", expired);
  }, []);

  if (!signedIn) return <Auth onLogin={() => setSignedIn(true)} />;

  return (
    <Workspace
      signOut={() => {
        void logout();
        setSignedIn(false);
      }}
      signedOut={() => setSignedIn(false)}
    />
  );
}
