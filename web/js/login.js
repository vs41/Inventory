document.getElementById("loginForm").addEventListener("submit", async (e) => {
  e.preventDefault();
  const err = document.getElementById("err");
  err.style.display = "none";
  try {
    const { role } = await API.login(
      document.getElementById("email").value,
      document.getElementById("password").value
    );
    window.location.href = role === "central_admin" ? "/admin/dashboard.html" : "/hub/scan.html";
  } catch (e2) {
    err.textContent = e2.message;
    err.style.display = "block";
  }
});
