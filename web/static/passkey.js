(function () {
  function csrfToken() {
    var input = document.querySelector('input[name="authenticity_token"]');
    return input ? input.value : "";
  }

  function setStatus(message) {
    var el = document.getElementById("passkey-status");
    if (el) {
      el.textContent = message || "";
    }
  }

  // WebAuthn JSON uses base64url (RFC 4648 §5): -/_ and no padding.
  // atob/btoa only speak standard base64 (+/ and =) and binary strings, not ArrayBuffers.
  function b64urlToBuf(value) {
    var padded = value.replace(/-/g, "+").replace(/_/g, "/");
    while (padded.length % 4) {
      padded += "=";
    }
    var binary = atob(padded);
    var bytes = new Uint8Array(binary.length);
    for (var i = 0; i < binary.length; i++) {
      bytes[i] = binary.charCodeAt(i);
    }
    return bytes.buffer;
  }

  function bufToB64url(buf) {
    var bytes = new Uint8Array(buf);
    var binary = "";
    for (var i = 0; i < bytes.length; i++) {
      binary += String.fromCharCode(bytes[i]);
    }
    return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
  }

  function prepareCreateOptions(publicKey) {
    var opts = Object.assign({}, publicKey);
    opts.challenge = b64urlToBuf(publicKey.challenge);
    opts.user = Object.assign({}, publicKey.user, {
      id: b64urlToBuf(publicKey.user.id),
    });
    if (publicKey.excludeCredentials) {
      opts.excludeCredentials = publicKey.excludeCredentials.map(function (cred) {
        return Object.assign({}, cred, { id: b64urlToBuf(cred.id) });
      });
    }
    return { publicKey: opts };
  }

  function credentialToJSON(cred) {
    var response = {
      clientDataJSON: bufToB64url(cred.response.clientDataJSON),
      attestationObject: bufToB64url(cred.response.attestationObject),
    };
    if (cred.response.getTransports) {
      response.transports = cred.response.getTransports();
    }
    return {
      id: cred.id,
      rawId: bufToB64url(cred.rawId),
      type: cred.type,
      response: response,
    };
  }

  function prepareGetOptions(publicKey) {
    var opts = Object.assign({}, publicKey);
    opts.challenge = b64urlToBuf(publicKey.challenge);
    if (publicKey.allowCredentials) {
      opts.allowCredentials = publicKey.allowCredentials.map(function (cred) {
        return Object.assign({}, cred, { id: b64urlToBuf(cred.id) });
      });
    }
    return { publicKey: opts };
  }

  function assertionToJSON(cred) {
    var response = {
      clientDataJSON: bufToB64url(cred.response.clientDataJSON),
      authenticatorData: bufToB64url(cred.response.authenticatorData),
      signature: bufToB64url(cred.response.signature),
    };
    if (cred.response.userHandle) {
      response.userHandle = bufToB64url(cred.response.userHandle);
    }
    return {
      id: cred.id,
      rawId: bufToB64url(cred.rawId),
      type: cred.type,
      response: response,
    };
  }

  async function parseError(res) {
    try {
      var data = await res.json();
      return data.error || res.statusText;
    } catch (e) {
      return res.statusText || "Request failed";
    }
  }

  async function add() {
    if (!window.PublicKeyCredential) {
      setStatus("Passkeys are not supported in this browser.");
      return;
    }
    setStatus("");
    try {
      var begin = await fetch("/web/passkey/register/begin", {
        method: "POST",
        headers: {
          Accept: "application/json",
          "X-CSRF-Token": csrfToken(),
        },
      });
      if (!begin.ok) {
        setStatus(await parseError(begin));
        return;
      }
      var options = await begin.json();
      var cred = await navigator.credentials.create(prepareCreateOptions(options.publicKey));
      var finish = await fetch("/web/passkey/register/finish", {
        method: "POST",
        headers: {
          Accept: "application/json",
          "Content-Type": "application/json",
          "X-CSRF-Token": csrfToken(),
        },
        body: JSON.stringify(credentialToJSON(cred)),
      });
      if (!finish.ok) {
        setStatus(await parseError(finish));
        return;
      }
      window.location.assign("/web/profile");
    } catch (err) {
      setStatus(err && err.message ? err.message : "Could not add passkey");
    }
  }

  async function login() {
    if (!window.PublicKeyCredential) {
      setStatus("Passkeys are not supported in this browser.");
      return;
    }
    var userIdInput = document.getElementById("user_id");
    var userId = userIdInput ? userIdInput.value.trim() : "";
    if (!userId) {
      setStatus("Enter your ID to use a passkey.");
      return;
    }
    setStatus("");
    try {
      var begin = await fetch("/web/passkey/login/begin", {
        method: "POST",
        headers: {
          Accept: "application/json",
          "Content-Type": "application/json",
          "X-CSRF-Token": csrfToken(),
        },
        body: JSON.stringify({ user_id: userId }),
      });
      if (!begin.ok) {
        setStatus(await parseError(begin));
        return;
      }
      var options = await begin.json();
      var cred = await navigator.credentials.get(prepareGetOptions(options.publicKey));
      var finish = await fetch("/web/passkey/login/finish", {
        method: "POST",
        headers: {
          Accept: "application/json",
          "Content-Type": "application/json",
          "X-CSRF-Token": csrfToken(),
        },
        body: JSON.stringify(assertionToJSON(cred)),
      });
      if (!finish.ok) {
        setStatus(await parseError(finish));
        return;
      }
      var result = await finish.json();
      window.location.assign(result.redirect);
    } catch (err) {
      setStatus(err && err.message ? err.message : "Could not sign in with passkey");
    }
  }

  function hideUnsupported() {
    if (window.PublicKeyCredential) {
      return;
    }
    var add = document.getElementById("add-passkey");
    if (add) {
      add.hidden = true;
    }
    var usePasskey = document.getElementById("use-passkey");
    if (usePasskey) {
      usePasskey.hidden = true;
    }
  }

  window.taPasskeys = { add: add, login: login };
  hideUnsupported();
})();
