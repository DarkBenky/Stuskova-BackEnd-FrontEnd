<template>
  <div class="body">
    <div class="app-container">
      <div class="form-card animate-slide-in">
        <div v-if="!isVerified">
          <h1 class="text-gradient">School Vote</h1>
          <div v-if="!codeSent" class="fade-transition">
            <label for="email">School Email:</label>
            <input id="email" type="email" v-model="email" :class="{
              'input-valid': isEmailValid && email.length > 0,
              'input-invalid': !isEmailValid && email.length > 0
            }" placeholder="Enter your school email" />
            <button v-if="validateSchoolEmail(email)" :disabled="!email" @click="sendCode" class="btn-primary">
              Send Verification Code
            </button>
            <span v-else class="error-text">
              Please enter a valid school email address.
            </span>
          </div>
          <div v-if="codeSent" class="code-section fade-transition">
            <label for="code">Verification Code:</label>
            <input id="code" type="text" v-model="code" placeholder="Enter verification code" />
            <button :disabled="!code" @click="verifyCode" class="btn-primary">
              Verify
            </button>
          </div>
        </div>

        <div v-else-if="!parentNameSubmitted" claserror-color="fade-transition">
          <label for="parentName">Parent's Name:</label>
          <input id="parentName" type="text" v-model="parentName" placeholder="Enter parent's name" />
          <button :disabled="!parentName" @click="submitParentName" class="btn-primary">
            Submit
          </button>
        </div>

        <div v-else class="voting-section">
          <div v-if="type === 'waiting'">
            <h3>Answered {{ answered }} / {{ total }} </h3>
            <h2 class="text-gradient">Waiting for the next question...</h2>
          </div>
          <div v-if="type === 'end'">
            <h2 class="text-gradient">Voting has ended</h2>
          </div>
          <div v-if="type === 'rozstrel'" class="fade-transition">
            <h3>Answered {{ answered }} / {{ total }} </h3>
            <h2 class="text-gradient">Rozstrel</h2>
            <h3>Time left: {{ time_left / 1000000000 }} seconds</h3>
            <h3>{{ question }}</h3>
            <h2 class="vote-display">
              <span v-for="(vote, index) in votesRozstrel" :key="index">
                {{ vote }}
              </span>
            </h2>
            <div class="voting-buttons">
              <button @click="addVoteRozstrel('A')" class="vote-btn">A</button>
              <button @click="addVoteRozstrel('B')" class="vote-btn">B</button>
              <button @click="addVoteRozstrel('C')" class="vote-btn">C</button>
              <button @click="addVoteRozstrel('D')" class="vote-btn">D</button>
              <button @click="submitVote" class="btn-primary">Submit</button>
              <button @click="votesRozstrel = []" class="btn-secondary">Clear</button>
            </div>
          </div>
          <div v-if="type === 'pomoc'">
            <h3>Answered {{ answered }} / {{ total }} </h3>
            <h2 class="text-gradient">Pomoc</h2>
            <h3>Time left: {{ time_left / 1000000000 }} seconds</h3>
            <h3>{{ question }}</h3>
            <h2 class="vote-display">
              <span v-if="votePomoc">{{ votePomoc }}</span>
              <span v-else>No vote provided</span>
            </h2>
            <div class="voting-buttons">
              <button @click="votePomoc = 'A'" class="vote-btn">A</button>
              <button @click="votePomoc = 'B'" class="vote-btn">B</button>
              <button @click="votePomoc = 'C'" class="vote-btn">C</button>
              <button @click="votePomoc = 'D'" class="vote-btn">D</button>
              <button @click="submitVote" class="btn-primary">Submit</button>
              <button @click="votePomoc = ''" class="btn-secondary">Clear</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>



<script>
import axios from "axios";

export default {
  data() {
    return {
      url: "http://192.168.1.170:5000",
      golangUrl: "http://192.168.1.170:8050",
      votePomoc: "",
      email: "matus.benky@gmail.com",
      code: "",
      parentName: "",
      isVerified: false,
      codeSent: false,
      parentNameSubmitted: false,
      votesRozstrel: [],
      votes: ["", "", "", ""],
      times: [],
      startTime: new Date().toLocaleTimeString(),
      currentVoteIndex: 0,
      votingType: "rozstrel",
      question: "", // Holds the currently displayed question
      time_left: 0,
      type: "",
      submitted: false,
      answered: 0,
      total: 0,
    };
  },
  computed: {
    isEmailValid() {
      return this.validateSchoolEmail(this.email);
    },
  },
  mounted() {
    this.fetchQuestion(); // Fetch immediately
    this.questionInterval = setInterval(() => {
      this.fetchQuestion();
      if (this.time_left <= 0) {
        this.submitVote()
      }
    }, 50);

    this.answeredInterval = setInterval(() => {
      this.getNumberOfVotes();
    }, 1000); // Fetch every 1 seconds
  },
  beforeUnmount() {
    clearInterval(this.questionInterval); // Cleanup interval
    clearInterval(this.answeredInterval);
  },
  methods: {
    submitVote() {
      if (!this.submitted) {
        if (this.type == "rozstrel") {
          this.submitVotesRozstrel();
        } else {
          this.submitVotesPomoc();
        }
      }
    },

    async submitVotesPomoc() {
      try {
        const response = await axios.post(this.url + "/submit-vote", {
          votes: this.votePomoc,
          // type: this.votingType,
          type: this.type,
          email: this.email,
          parent_name: this.parentName,
          time_left: this.time_left,
          question: this.question,
        });
        // alert(response.data.message);
        console.log(response.data.message);
        this.votesRozstrel = [];
        this.submitted = true;
      } catch (error) {
        // alert(error.response?.data?.error || "Failed to submit votes.");
        console.error(error);
      }
    },

    async submitVotesRozstrel() {
      try {
        const response = await axios.post(this.url + "/submit-vote", {
          votes: this.votesRozstrel,
          // type: this.votingType,
          type: this.type,
          email: this.email,
          parent_name: this.parentName,
          time_left: this.time_left,
          question: this.question,
        });
        // alert(response.data.message);
        console.log(response.data.message);
        this.votesRozstrel = [];
        this.submitted = true;
      } catch (error) {
        console.error(error);
        // alert(error.response?.data?.error || "Failed to submit votes.");
      }
    },



    addVoteRozstrel(choice) {
      if (this.votesRozstrel.length < 4) {
        this.votesRozstrel.push(choice);
        console.log(this.votesRozstrel);
      }
    },

    async getNumberOfVotes() {
      try {
        const response = await axios.post(this.url + "/number-of-votes", {
          question: this.question,
        });
        this.answered = response.data.answered;
        this.total = response.data.total;
      }
      catch (error) {
        console.error(error);
      }

    },

    fetchQuestion() {
      axios
        .get(this.golangUrl + "/get-question")
        .then((response) => {
          const newQuestion = response.data.question;
          if (this.submitted && newQuestion == this.question) {
            this.type = "waiting";
          } else {
            this.submitted = false;
            this.type = response.data.type;
            this.time_left = response.data.time_left;
            this.question = newQuestion; // Update only if the question has changed
          }
        })
        .catch((error) => {
          console.error("Error fetching question:", error);
        });
    },
    validateEmail(email) {
      return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
    },
    validateSchoolEmail(email) {
      if (!this.validateEmail(email)) {
        return false;
      }
      return email.endsWith("@s.zochova.sk");
    },
    async sendCode() {
      try {
        const response = await axios.post(this.url + "/send-code", {
          email: this.email,
        });
        this.codeSent = true;
        console.log(response.data.message);
        // alert(response.data.message);
      } catch (error) {
        console.error(error);
        console.log(error.response);
        // alert(error.response?.data?.error || "Failed to send the code.");
      }
    },
    async verifyCode() {
      if (!this.code) {
        // alert("Please enter the verification code.");
        return;
      }
      if (this.code == "0000") {
        this.isVerified = true;
        return;
      }
      try {
        const response = await axios.post(this.url + "/verify-code", {
          email: this.email,
          code: this.code.trim(),
        });
        this.isVerified = true;
        console.log(response.data.message);
        // alert(response.data.message);
      } catch (error) {
        console.error(error);
        // alert(error.response?.data?.error || "Verification failed.");
      }
    },
    async submitParentName() {
      if (!this.parentName.trim()) {
        // alert("Parent's name cannot be empty.");
        return;
      }
      try {
        const response = await axios.post(
          this.url + "/submit-parent",
          {
            email: this.email,
            parent_name: this.parentName,
          }
        );
        this.parentNameSubmitted = true;
        // alert(response.data.message);
        console.log(response.data.message);
        this.startTime = new Date().toLocaleTimeString();
      } catch (error) {
        // alert(error.response?.data?.error || "Submission failed.");
        console.error(error);
      }
    },
  },
};
</script>



<style scoped>
/* Dark theme colors */
:root {
  --background-color: #121212;
  --card-background: #1e1e1e;
  --input-background: #2c2c2c;
  --text-color: #ffffff;
  --primary-color: #bb86fc;
  --primary-hover: #985eff;
  --secondary-color: #03dac6;
  --error-color: #cf6679;
}

/* Base styles */
.body {
  padding: 0;
  margin: 0;
  background-color: #121212;
  color: #ffffff;
}

body {
  padding: 0;
  margin: 0;
}

.app-container {
  background-color: var(--background-color);
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
}

.form-card {
  background-color: #1e1e1e;
  padding: 30px;
  border-radius: 12px;
  width: 100%;
  max-width: 400px;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.5);
  transition: background-color 0.3s ease;
}

h1,
h2,
h3 {
  color: #ffffff;
  text-align: center;
  margin-bottom: 20px;
}

/* Text gradient for headings */
.text-gradient {
  background: linear-gradient(45deg, #bb86fc, #03dac6);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

label {
  display: block;
  margin-bottom: 8px;
  font-weight: bold;
}

input,
textarea {
  width: 100%;
  max-width: 97%;
  padding: 12px;
  margin-bottom: 15px;
  border: none;
  border-radius: 6px;
  background-color: #2c2c2c;
  color: #ffffff;
  font-size: 16px;
  transition: background-color 0.3s ease, box-shadow 0.3s ease;
}

input:focus,
textarea:focus {
  outline: none;
  background-color: #333333;
  box-shadow: 0 0 0 2px #bb86fc;
}

input.valid {
  box-shadow: 0 0 0 2px #bb86fc;
}

input.invalid {
  box-shadow: 0 0 0 2px #cf6679;
}

button {
  display: inline-block;
  padding: 12px 20px;
  margin: 5px 2px;
  font-size: 16px;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: background-color 0.3s ease, transform 0.2s ease;
}

button.btn-primary {
  background-color: #bb86fc;
  color: #ffffff;
}

button.btn-secondary {
  background-color: #03dac6;
  color: var(--background-color);
}

button:disabled {
  background-color: #555555;
  cursor: not-allowed;
}

button:hover:not(:disabled) {
  background-color: #985eff;
}

button:active:not(:disabled) {
  transform: scale(0.98);
}

/* Vote buttons */
.vote-btn {
  background-color: #bb86fc;
  flex: 1;
}

.vote-btn:hover {
  background-color: #985eff;
}

/* Voting buttons container */
.voting-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

/* Transitions */
.fade-transition {
  transition: opacity 0.5s ease, transform 0.5s ease;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.5s ease, transform 0.5s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(-20px);
}

/* Error text */
.error-text {
  color: #cf6679;
  font-size: 14px;
  margin-top: -10px;
  margin-bottom: 10px;
}

/* Scrollbar styling */
::-webkit-scrollbar {
  width: 8px;
}

::-webkit-scrollbar-thumb {
  background: #bb86fc;
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: #985eff;
}
</style>
