<script setup lang="ts">
// 占位组件
</script>

<template>
  <div class="home-container">
    <div class="placeholder-content">
      <el-empty description="功能开发中">
        <template #image>
          <el-icon size="80">
            <Tools />
          </el-icon>
        </template>
        <el-button type="primary">敬请期待</el-button>
      </el-empty>
    </div>
  </div>
</template>

<style scoped lang="scss">
.home-container {
  height: 100%;
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.placeholder-content {
  text-align: center;
}
</style>