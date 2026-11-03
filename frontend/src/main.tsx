import ReactDOM from "react-dom/client";
import "@wailsio/runtime";
import { App } from "./App";
import { Tray } from "./Tray";
import "./style.css";

const tray = location.hash === "#tray";
if (tray) document.body.classList.add("tray-body");
ReactDOM.createRoot(document.getElementById("root")!).render(
  tray ? <Tray /> : <App />,
);
