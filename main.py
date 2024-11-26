from flask import Flask, request, jsonify, session
import smtplib
import random
from email.mime.text import MIMEText
from flask_cors import CORS

app = Flask(__name__)
app.secret_key = 'your_secret_key'
CORS(app)  # Allow cross-origin requests

# Email Configuration
SENDER_EMAIL = "matus.benky@gmail.com"
EMAIL_PASSWORD = "qnaq uomr pmva jajp"

# Temporary storage for verification codes and parent names
verification_codes = {}
parent_names = {}

def send_email(subject, body, sender, recipient, password):
    """
    Sends an email using SMTP.
    """
    try:
        msg = MIMEText(body)
        msg['Subject'] = subject
        msg['From'] = sender
        msg['To'] = recipient

        with smtplib.SMTP_SSL('smtp.gmail.com', 465) as smtp_server:
            smtp_server.login(sender, password)
            smtp_server.sendmail(sender, recipient, msg.as_string())
        print(f"Email sent to {recipient}")
    except Exception as e:
        print(f"Failed to send email: {e}")


@app.route('/send-code', methods=['POST'])
def send_code():
    """Send a verification code to the user's email."""
    email = request.json.get('email')
    verification_code = random.randint(100000, 999999)
    verification_codes[email] = verification_code

    send_email(
        "Your Verification Code",
        f"Your verification code is {verification_code}",
        SENDER_EMAIL,
        email,
        EMAIL_PASSWORD
    )
    return jsonify({"message": "Verification code sent successfully!"})


@app.route('/verify-code', methods=['POST'])
def verify_code():
    """Verify the user's email using the code."""
    email = request.json.get('email')
    code = request.json.get('code')

    if not email or not code:
        return jsonify({"error": "Email and code are required"}), 400

    try:
        code = int(code)
    except ValueError:
        return jsonify({"error": "Invalid code format"}), 400

    stored_code = verification_codes.get(email)
    if not stored_code:
        return jsonify({"error": "No verification code found for this email"}), 400

    if stored_code == code:
        # Optional: Clear the code after successful verification
        verification_codes.pop(email)
        return jsonify({"message": "Verification successful!"})
    
    return jsonify({"error": "Invalid verification code"}), 400


@app.route('/submit-parent', methods=['POST'])
def submit_parent():
    """Save parent's name."""
    email = request.json.get('email')
    parent_name = request.json.get('parent_name')
    parent_names[email] = {"parent_name": parent_name, "email": email , "votes": {}}
    return jsonify({"message": "Parent name submitted successfully!"})

@app.route('/get-question', methods=['GET'])
def get_qeustion():
    return jsonify({
        "question": random.choice(["Asadadad", "Badadad", "Cadadad", "Dadaddad"]),
        "time_left": random.randint(1, 10),
        # "type": random.choice(["rozstrel", "pomoc","waiting", "end"]),
        # "type": random.choice(["rozstrel"]),
        "type": random.choice(["pomoc"]),
        })

def log(message):
    print(message)

@app.route('/number-of-votes', methods=['POST'])
def number_of_votes():
    current_question = request.get_json().get('question')

    answered = 0

    for email in parent_names:
        if parent_names[email]["votes"].get(current_question):
            answered += 1
    
    return jsonify({"answered": answered, "total": len(parent_names)})

@app.route('/submit-vote', methods=['POST'])
def submit_vote():
    try:
        data = request.get_json()
        
        # Extract data from request
        votes = data.get('votes')
        vote_type = data.get('type')
        email = data.get('email')
        time_left = data.get('time_left')
        question = data.get('question')
        parent_name = parent_names[email]["parent_name"]

        log(f"Question: {question}")

        # Validate required fields
        if votes == '' or len(votes) == 0:
            log("Missing required fields")
            return jsonify({'error': 'Missing required fields'}), 400
        
        # check if parent did vote
        if parent_names[email]["votes"].get(question):
            log("Parent already voted")
            log(parent_names[email]["votes"])
            return jsonify({'error': 'Parent already voted'}), 400
        
        # parent_names[email]["votes"][question] = {"vote":votes, "type":vote_type, "time_left":time_left}
        parent_names[email]["votes"][question] = {'vote':votes, 'type':vote_type, 'time_left':time_left}

        print(parent_names)

        # Store vote in database or processing logic here
        # For now, just return success message
        return jsonify({
            'message': 'Vote submitted successfully',
            'data': {
                'votes': votes,
                'type': vote_type,
                'email': email,
                'parent_name': parent_name,
                'time_left': time_left
            }
        }), 200

    except Exception as e:
        log(f"An error occurred: {str(e)}")
        return jsonify({'error': str(e)}), 500
    


if __name__ == '__main__':
    app.run(debug=True, host='0.0.0.0', port=5000)
