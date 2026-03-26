(() => {
    "use strict";

    const $ = (sel) => document.querySelector(sel);
    const statusEl = $("#status");
    const gridEl = $("#agent-grid");
    const emptyEl = $("#empty-msg");
    const gridView = $("#grid-view");
    const controlView = $("#control-view");
    const canvas = $("#remote-canvas");
    const ctx = canvas.getContext("2d");
    const agentNameEl = $("#control-agent-name");
    const fpsEl = $("#control-fps");

    let ws = null;
    let browserID = null;
    const agents = new Map(); // id -> {info, thumbEl, imgEl}

    // Current control session
    let currentAgentID = null;
    let peerConnection = null;
    let videoChannel = null;
    let controlChannel = null;
    let remoteWidth = 1920;
    let remoteHeight = 1080;
    let frameCount = 0;

    // ---- WebSocket ----

    function connect() {
        const proto = location.protocol === "https:" ? "wss:" : "ws:";
        ws = new WebSocket(`${proto}//${location.host}/ws/browser`);

        ws.onopen = () => {
            statusEl.textContent = "connected";
            statusEl.className = "status connected";
        };

        ws.onclose = () => {
            statusEl.textContent = "disconnected";
            statusEl.className = "status disconnected";
            browserID = null;
            setTimeout(connect, 3000);
        };

        ws.onmessage = (e) => {
            const msg = JSON.parse(e.data);
            handleWSMessage(msg);
        };
    }

    function sendWS(msg) {
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify(msg));
        }
    }

    function handleWSMessage(msg) {
        switch (msg.type) {
            case "registered":
                browserID = msg.browser_id;
                break;

            case "agents":
                for (const a of msg.agents) addAgent(a);
                updateEmpty();
                break;

            case "agent_joined":
                addAgent(msg.agent);
                updateEmpty();
                break;

            case "agent_left":
                removeAgent(msg.agent_id);
                updateEmpty();
                if (currentAgentID === msg.agent_id) exitControl();
                break;

            case "thumbnail":
                updateThumbnail(msg.agent_id, msg.data);
                break;

            case "answer":
                if (peerConnection) {
                    peerConnection.setRemoteDescription({
                        type: "answer",
                        sdp: msg.sdp,
                    });
                }
                break;

            case "candidate":
                if (peerConnection && msg.candidate) {
                    peerConnection.addIceCandidate(msg.candidate);
                }
                break;

            case "error":
                console.error("Server error:", msg.data);
                break;
        }
    }

    // ---- Agent Grid ----

    function addAgent(info) {
        if (agents.has(info.id)) return;

        const card = document.createElement("div");
        card.className = "agent-card";
        card.dataset.agentId = info.id;

        const thumbContainer = document.createElement("div");
        thumbContainer.className = "thumb-container";

        const img = document.createElement("img");
        img.src = "data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7";
        img.alt = info.name;
        thumbContainer.appendChild(img);

        const infoDiv = document.createElement("div");
        infoDiv.className = "agent-info";
        infoDiv.innerHTML = `<span>${escapeHtml(info.name)}</span><span class="os">${escapeHtml(info.os)} ${info.width}x${info.height}</span>`;

        card.appendChild(thumbContainer);
        card.appendChild(infoDiv);

        card.addEventListener("click", () => enterControl(info));

        gridEl.appendChild(card);
        agents.set(info.id, { info, card, img });
    }

    function removeAgent(id) {
        const agent = agents.get(id);
        if (agent) {
            agent.card.remove();
            agents.delete(id);
        }
    }

    function updateThumbnail(id, b64) {
        const agent = agents.get(id);
        if (agent) {
            agent.img.src = "data:image/jpeg;base64," + b64;
        }
    }

    function updateEmpty() {
        emptyEl.classList.toggle("show", agents.size === 0);
    }

    // ---- WebRTC Control ----

    async function enterControl(agentInfo) {
        currentAgentID = agentInfo.id;
        agentNameEl.textContent = `${agentInfo.name} (${agentInfo.os})`;
        gridView.style.display = "none";
        controlView.style.display = "flex";

        const config = {
            iceServers: [{ urls: "stun:stun.l.google.com:19302" }],
        };
        peerConnection = new RTCPeerConnection(config);

        // Create data channels
        videoChannel = peerConnection.createDataChannel("video", { ordered: false, maxRetransmits: 0 });
        videoChannel.binaryType = "arraybuffer";
        videoChannel.onmessage = onVideoFrame;

        controlChannel = peerConnection.createDataChannel("control");
        controlChannel.onmessage = (e) => {
            const msg = JSON.parse(e.data);
            if (msg.type === "screen_info") {
                remoteWidth = msg.width;
                remoteHeight = msg.height;
            }
        };

        peerConnection.onicecandidate = (e) => {
            if (e.candidate) {
                sendWS({
                    type: "candidate",
                    agent_id: currentAgentID,
                    candidate: e.candidate.toJSON(),
                });
            }
        };

        const offer = await peerConnection.createOffer();
        await peerConnection.setLocalDescription(offer);

        sendWS({
            type: "offer",
            agent_id: currentAgentID,
            sdp: offer.sdp,
        });

        // FPS counter
        frameCount = 0;
        fpsInterval = setInterval(() => {
            fpsEl.textContent = frameCount + " FPS";
            frameCount = 0;
        }, 1000);

        // Bind input events
        bindInputEvents();
    }

    let fpsInterval = null;

    function exitControl() {
        gridView.style.display = "block";
        controlView.style.display = "none";
        currentAgentID = null;

        if (peerConnection) {
            peerConnection.close();
            peerConnection = null;
        }
        videoChannel = null;
        controlChannel = null;

        if (fpsInterval) {
            clearInterval(fpsInterval);
            fpsInterval = null;
        }
        unbindInputEvents();
    }

    $("#btn-back").addEventListener("click", exitControl);

    // ---- Video rendering ----

    const frameImage = new Image();
    frameImage.onload = () => {
        canvas.width = frameImage.naturalWidth;
        canvas.height = frameImage.naturalHeight;
        ctx.drawImage(frameImage, 0, 0);
        frameCount++;
    };

    function onVideoFrame(e) {
        const blob = new Blob([e.data], { type: "image/jpeg" });
        const url = URL.createObjectURL(blob);
        // Revoke previous URL to avoid memory leak
        if (frameImage._prevURL) URL.revokeObjectURL(frameImage._prevURL);
        frameImage._prevURL = url;
        frameImage.src = url;
    }

    // ---- Input forwarding ----

    function getRelativePos(e) {
        const rect = canvas.getBoundingClientRect();
        const scaleX = canvas.width / rect.width;
        const scaleY = canvas.height / rect.height;
        const px = (e.clientX - rect.left) * scaleX;
        const py = (e.clientY - rect.top) * scaleY;
        return {
            x: px / canvas.width,
            y: py / canvas.height,
        };
    }

    function sendControl(msg) {
        if (controlChannel && controlChannel.readyState === "open") {
            controlChannel.send(JSON.stringify(msg));
        }
    }

    function onMouseMove(e) {
        const pos = getRelativePos(e);
        sendControl({ type: "mousemove", ...pos });
    }

    function onMouseDown(e) {
        e.preventDefault();
        const pos = getRelativePos(e);
        sendControl({ type: "mousedown", ...pos, button: e.button });
    }

    function onMouseUp(e) {
        e.preventDefault();
        const pos = getRelativePos(e);
        sendControl({ type: "mouseup", ...pos, button: e.button });
    }

    function onWheel(e) {
        e.preventDefault();
        const pos = getRelativePos(e);
        sendControl({ type: "scroll", ...pos, deltaX: e.deltaX, deltaY: e.deltaY });
    }

    function onKeyDown(e) {
        e.preventDefault();
        sendControl({ type: "keydown", key: e.key, code: e.code });
    }

    function onKeyUp(e) {
        e.preventDefault();
        sendControl({ type: "keyup", key: e.key, code: e.code });
    }

    function onContextMenu(e) { e.preventDefault(); }

    function bindInputEvents() {
        canvas.addEventListener("mousemove", onMouseMove);
        canvas.addEventListener("mousedown", onMouseDown);
        canvas.addEventListener("mouseup", onMouseUp);
        canvas.addEventListener("wheel", onWheel, { passive: false });
        canvas.addEventListener("contextmenu", onContextMenu);
        document.addEventListener("keydown", onKeyDown);
        document.addEventListener("keyup", onKeyUp);
    }

    function unbindInputEvents() {
        canvas.removeEventListener("mousemove", onMouseMove);
        canvas.removeEventListener("mousedown", onMouseDown);
        canvas.removeEventListener("mouseup", onMouseUp);
        canvas.removeEventListener("wheel", onWheel);
        canvas.removeEventListener("contextmenu", onContextMenu);
        document.removeEventListener("keydown", onKeyDown);
        document.removeEventListener("keyup", onKeyUp);
    }

    // ---- Utils ----

    function escapeHtml(str) {
        const div = document.createElement("div");
        div.textContent = str;
        return div.innerHTML;
    }

    // ---- Init ----
    connect();
})();
