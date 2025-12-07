<template>
   
    <nav class="navbar" role="navigation" aria-label="main navigation">
      <div class="navbar-brand">
        <h3 class="logo">Macro Sentiment</h3>
        <a role="button" class="navbar-burger" aria-label="menu" aria-expanded="false" data-target="navbarBasicExample">
          <span aria-hidden="true"></span>
          <span aria-hidden="true"></span>
          <span aria-hidden="true"></span>
          <span aria-hidden="true"></span>
        </a>
      </div>
    </nav>
</template>

<style lang="scss" scoped>
@import "../assets/scss/main";

nav {
  background-color: $primary-color !important;
  height: 50px !important;
}


.logo {
  color: white !important;
  padding-top: 12px;
  padding-left: 12px;
  font-family: $primary-font;
}
</style>