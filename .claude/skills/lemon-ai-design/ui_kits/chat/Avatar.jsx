// Avatar.jsx — the round profile picture shown next to each message.
// One per role (user / assistant). Clicking it lets you set a real picture;
// the chosen image persists in localStorage (see useProfiles in App.jsx).
//
// Fallbacks when no picture is set:
//   assistant → the lemon mark (brand asset)
//   user      → a quiet person glyph on paper
//
// `editable` turns the avatar into a button that opens a file picker.

function Avatar({ role, profile, size = 30, editable = true, onSetPic }) {
  const fileRef = React.useRef();
  const isUser = role === "user";
  const src = profile?.src;
  const label = isUser ? "your picture" : "lemon's picture";

  const pick = () => fileRef.current && fileRef.current.click();

  const onFile = (e) => {
    const file = e.target.files && e.target.files[0];
    e.target.value = ""; // allow re-picking the same file
    if (!file || !file.type.startsWith("image/")) return;
    const reader = new FileReader();
    reader.onload = () => onSetPic && onSetPic(reader.result);
    reader.readAsDataURL(file);
  };

  const inner = src ? (
    <img src={src} alt="" draggable="false" />
  ) : isUser ? (
    <span className="avatar-glyph"><Icon name="user" size={Math.round(size * 0.56)} /></span>
  ) : (
    <img className="avatar-mark" src="../../assets/logo-mark.svg" alt="" draggable="false" />
  );

  const cls = "avatar " + role + (src ? " has-pic" : "");
  const style = { width: size, height: size };

  if (!editable) {
    return <div className={cls} style={style} aria-hidden="true">{inner}</div>;
  }

  return (
    <button
      type="button"
      className={cls}
      style={style}
      onClick={pick}
      title={"Change " + label}
      aria-label={"Change " + label}
    >
      {inner}
      <span className="avatar-cam" aria-hidden="true">
        <Icon name="camera" size={Math.round(size * 0.42)} />
      </span>
      <input
        ref={fileRef}
        type="file"
        accept="image/*"
        onChange={onFile}
        style={{ display: "none" }}
      />
    </button>
  );
}

window.Avatar = Avatar;
