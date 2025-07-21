/* Workflow-Specific Help Guides - Step-by-Step Instructions
   Tailored for older and non-technical users */

class WorkflowGuides {
  constructor() {
    this.guides = {
      // Data Import Workflows
      'import-ecse': {
        title: 'Import ECSE Student Data',
        icon: 'file-earmark-medical',
        description: 'Learn how to import special education student data from Excel files',
        steps: [
          {
            title: 'Prepare Your Excel File',
            description: 'Make sure your Excel file has the correct columns: Student ID, First Name, Last Name, Date of Birth, Grade, and other required fields.',
            tip: 'Download our template to ensure your data is formatted correctly.',
            warning: 'Save your file as .xlsx format - older .xls files may not work properly.'
          },
          {
            title: 'Click Browse to Select File',
            description: 'Click the "Browse" or "Choose File" button to open your computer\'s file picker.',
            tip: 'Look for files ending in .xlsx in your Downloads or Documents folder.',
            warning: null
          },
          {
            title: 'Review Data Preview',
            description: 'After selecting your file, review the preview to make sure all data looks correct.',
            tip: 'Check that names and dates appear properly formatted.',
            warning: 'If data looks scrambled, your file may have formatting issues.'
          },
          {
            title: 'Click Import Button',
            description: 'Once you\'ve reviewed the data, click the "Import" button to add the students to the system.',
            tip: 'This process may take a few moments for large files.',
            warning: 'Don\'t close the browser window while importing - you may lose your data.'
          },
          {
            title: 'Verify Import Success',
            description: 'Check the results page to confirm all students were imported successfully.',
            tip: 'Any errors will be clearly shown so you can fix them.',
            warning: 'If there are errors, you can download a corrected file and try again.'
          }
        ]
      },
      
      'import-mileage': {
        title: 'Import Mileage Reports',
        icon: 'speedometer2',
        description: 'Import monthly mileage data from your spreadsheet',
        steps: [
          {
            title: 'Gather Your Mileage Data',
            description: 'Collect your monthly mileage reports with columns for Bus ID, Date, Starting Mileage, Ending Mileage, and Driver.',
            tip: 'Use our template to ensure all required fields are included.',
            warning: 'Make sure all dates are in MM/DD/YYYY format.'
          },
          {
            title: 'Select Your Excel File',
            description: 'Click "Browse" and find your mileage spreadsheet on your computer.',
            tip: 'Files are usually in your Downloads folder after saving from email.',
            warning: 'Only .xlsx files are accepted - convert older Excel files first.'
          },
          {
            title: 'Review Mileage Preview',
            description: 'Check that all mileage numbers and dates appear correctly in the preview.',
            tip: 'Look for any obviously wrong numbers (like negative mileage).',
            warning: 'Incorrect mileage data can affect fuel cost calculations.'
          },
          {
            title: 'Complete the Import',
            description: 'Click "Import Mileage Data" to add all records to the system.',
            tip: 'Large files may take several minutes to process.',
            warning: 'Don\'t navigate away from the page during import.'
          }
        ]
      },
      
      // Fleet Management Workflows
      'add-bus': {
        title: 'Add New Bus to Fleet',
        icon: 'bus-front',
        description: 'Register a new bus in the fleet management system',
        steps: [
          {
            title: 'Enter Bus Information',
            description: 'Fill in the Bus ID (like "Bus 101"), model, and capacity information.',
            tip: 'Use a consistent naming system like "Bus ###" for easy identification.',
            warning: 'Each Bus ID must be unique - you can\'t use the same number twice.'
          },
          {
            title: 'Set Initial Status',
            description: 'Choose whether the bus is Active, Under Maintenance, or Out of Service.',
            tip: 'Most new buses should be set to "Active" status.',
            warning: 'Buses marked "Out of Service" won\'t appear in route assignments.'
          },
          {
            title: 'Add Maintenance Information',
            description: 'Enter current oil status, tire condition, and any maintenance notes.',
            tip: 'Use simple terms like "Good", "Fair", or "Needs Attention".',
            warning: 'Be specific about any safety issues in the maintenance notes.'
          },
          {
            title: 'Save Bus Information',
            description: 'Click "Add Bus" to save the new vehicle to your fleet.',
            tip: 'You can always edit this information later if needed.',
            warning: 'Double-check all information before saving - some fields can\'t be changed easily.'
          }
        ]
      },
      
      'update-maintenance': {
        title: 'Update Bus Maintenance',
        icon: 'wrench',
        description: 'Record maintenance work and update bus status',
        steps: [
          {
            title: 'Find Your Bus',
            description: 'Use the search box to find the bus you want to update, or browse the fleet list.',
            tip: 'You can search by Bus ID, model, or status.',
            warning: 'Make sure you\'re updating the correct bus before making changes.'
          },
          {
            title: 'Click Edit or Maintenance',
            description: 'Click the "Edit" button or "Maintenance" link next to the bus.',
            tip: 'Look for the wrench icon or "Edit" button in the actions column.',
            warning: 'Some buses may be locked for editing if they\'re currently on routes.'
          },
          {
            title: 'Update Status Information',
            description: 'Change the oil status, tire status, or overall bus condition as needed.',
            tip: 'Use consistent terms so other staff can understand the status.',
            warning: 'If marking a bus as "Out of Service", add a detailed explanation.'
          },
          {
            title: 'Add Maintenance Notes',
            description: 'Describe what work was done, when it was completed, and any follow-up needed.',
            tip: 'Include dates, costs, and who performed the work.',
            warning: 'These notes help track maintenance history and warranty information.'
          },
          {
            title: 'Save Changes',
            description: 'Click "Save" or "Update" to record the maintenance information.',
            tip: 'The bus status will update immediately in the system.',
            warning: 'Status changes affect route assignments and driver schedules.'
          }
        ]
      },
      
      // Student Management Workflows
      'add-student': {
        title: 'Add New Student',
        icon: 'person-plus',
        description: 'Register a new student in the transportation system',
        steps: [
          {
            title: 'Enter Student Details',
            description: 'Fill in the student\'s name, ID number, and grade level.',
            tip: 'Use the same ID format as your school\'s student information system.',
            warning: 'Student IDs must be unique - check if the ID already exists.'
          },
          {
            title: 'Add Contact Information',
            description: 'Enter parent/guardian names and phone numbers for emergency contact.',
            tip: 'Include both home and cell phone numbers when possible.',
            warning: 'This information is critical for safety and emergency situations.'
          },
          {
            title: 'Set Pickup and Dropoff',
            description: 'Enter the student\'s pickup location, dropoff location, and scheduled times.',
            tip: 'Be specific about addresses and any special instructions.',
            warning: 'Incorrect addresses can cause delays and safety issues.'
          },
          {
            title: 'Assign to Route',
            description: 'Select the appropriate bus route and assign a position number.',
            tip: 'Position numbers help drivers know the order of stops.',
            warning: 'Make sure the route serves the student\'s pickup and dropoff locations.'
          },
          {
            title: 'Save Student Information',
            description: 'Review all information and click "Add Student" to save.',
            tip: 'You can edit this information later if family circumstances change.',
            warning: 'Double-check all contact information for accuracy.'
          }
        ]
      },
      
      // Route Management Workflows
      'assign-route': {
        title: 'Assign Driver to Route',
        icon: 'diagram-3',
        description: 'Assign a driver and bus to a specific route',
        steps: [
          {
            title: 'Select Driver',
            description: 'Choose an available driver from the dropdown list.',
            tip: 'Only active drivers with proper certifications will appear.',
            warning: 'Make sure the driver is available for the route\'s schedule.'
          },
          {
            title: 'Choose Bus',
            description: 'Select an available bus that\'s appropriate for the route.',
            tip: 'Consider the number of students and any special needs.',
            warning: 'Buses under maintenance or out of service can\'t be assigned.'
          },
          {
            title: 'Select Route',
            description: 'Pick the route that needs a driver and bus assignment.',
            tip: 'Review the route details to understand the stops and timing.',
            warning: 'One driver can only be assigned to one route at a time.'
          },
          {
            title: 'Set Assignment Date',
            description: 'Enter when this assignment should start (usually the next school day).',
            tip: 'Most assignments start on Monday for the full week.',
            warning: 'You can\'t assign routes for past dates.'
          },
          {
            title: 'Confirm Assignment',
            description: 'Review the assignment details and click "Assign Route".',
            tip: 'The driver will see this assignment on their dashboard.',
            warning: 'Once assigned, the driver and bus become unavailable for other routes.'
          }
        ]
      },
      
      // Daily Operations Workflows
      'driver-log': {
        title: 'Complete Daily Driver Log',
        icon: 'journal-text',
        description: 'Record daily trip information and student attendance',
        steps: [
          {
            title: 'Select Your Route',
            description: 'Choose your assigned route from the dropdown menu.',
            tip: 'Your route should appear automatically if you\'re logged in.',
            warning: 'If you don\'t see your route, contact your supervisor.'
          },
          {
            title: 'Choose Time Period',
            description: 'Select whether this is your Morning or Afternoon route.',
            tip: 'Complete separate logs for morning and afternoon runs.',
            warning: 'Don\'t mix morning and afternoon information in one log.'
          },
          {
            title: 'Enter Trip Times',
            description: 'Record your departure time from the depot and arrival time at school.',
            tip: 'Use the format shown (like 7:30 AM) for consistency.',
            warning: 'Accurate times help with route planning and parent communication.'
          },
          {
            title: 'Record Student Attendance',
            description: 'Mark which students were present and note any pickup times.',
            tip: 'This helps track student patterns and safety.',
            warning: 'Always mark students as present or absent - never leave blank.'
          },
          {
            title: 'Enter Mileage',
            description: 'Record your starting and ending mileage for the trip.',
            tip: 'Check your odometer before starting and after completing the route.',
            warning: 'Accurate mileage is required for fuel reporting and maintenance.'
          },
          {
            title: 'Save Your Log',
            description: 'Review all information and click "Save Log" to complete your entry.',
            tip: 'You can edit the log later if you notice any mistakes.',
            warning: 'Some systems require logs to be submitted within 24 hours.'
          }
        ]
      }
    };
  }

  // Get guide by ID
  getGuide(guideId) {
    return this.guides[guideId] || null;
  }

  // Get all guides
  getAllGuides() {
    return this.guides;
  }

  // Get guides by category
  getGuidesByCategory(category) {
    const categoryGuides = {};
    
    Object.entries(this.guides).forEach(([id, guide]) => {
      if (id.startsWith(category)) {
        categoryGuides[id] = guide;
      }
    });
    
    return categoryGuides;
  }

  // Create HTML for a guide
  createGuideHTML(guideId) {
    const guide = this.getGuide(guideId);
    if (!guide) return '';
    
    let html = `
      <div class="workflow-guide" data-guide-id="${guideId}">
        <div class="workflow-guide-header">
          <div class="workflow-guide-icon">
            <i class="bi bi-${guide.icon}" aria-hidden="true"></i>
          </div>
          <div class="workflow-guide-title">
            <h3>${guide.title}</h3>
            <p>${guide.description}</p>
          </div>
        </div>
        
        <div class="workflow-guide-content">
          <ol class="workflow-steps">
    `;
    
    guide.steps.forEach((step, index) => {
      html += `
        <li class="workflow-step">
          <div class="workflow-step-number">${index + 1}</div>
          <div class="workflow-step-content">
            <h4 class="workflow-step-title">${step.title}</h4>
            <p class="workflow-step-description">${step.description}</p>
            
            ${step.tip ? `
              <div class="workflow-tip">
                <i class="bi bi-lightbulb" aria-hidden="true"></i>
                <strong>Tip:</strong> ${step.tip}
              </div>
            ` : ''}
            
            ${step.warning ? `
              <div class="workflow-warning">
                <i class="bi bi-exclamation-triangle" aria-hidden="true"></i>
                <strong>Important:</strong> ${step.warning}
              </div>
            ` : ''}
          </div>
        </li>
      `;
    });
    
    html += `
          </ol>
        </div>
      </div>
    `;
    
    return html;
  }

  // Add guide to page
  addGuideToPage(guideId, targetElement) {
    const guide = this.getGuide(guideId);
    if (!guide) return null;
    
    const guideHTML = this.createGuideHTML(guideId);
    const guideElement = document.createElement('div');
    guideElement.innerHTML = guideHTML;
    
    if (targetElement) {
      targetElement.appendChild(guideElement.firstElementChild);
    } else {
      document.body.appendChild(guideElement.firstElementChild);
    }
    
    return guideElement.firstElementChild;
  }

  // Create guide selector
  createGuideSelector(category = null) {
    const guides = category ? this.getGuidesByCategory(category) : this.getAllGuides();
    
    let html = `
      <div class="workflow-guide-selector">
        <h3>
          <i class="bi bi-question-circle" aria-hidden="true"></i>
          How-To Guides
        </h3>
        <p>Choose a guide to get step-by-step instructions:</p>
        
        <div class="guide-selector-grid">
    `;
    
    Object.entries(guides).forEach(([id, guide]) => {
      html += `
        <button class="guide-selector-btn" data-guide-id="${id}">
          <i class="bi bi-${guide.icon}" aria-hidden="true"></i>
          <span>${guide.title}</span>
        </button>
      `;
    });
    
    html += `
        </div>
      </div>
    `;
    
    return html;
  }

  // Initialize guide selector on page
  initializeGuideSelector(targetElement, category = null) {
    const selectorHTML = this.createGuideSelector(category);
    const selectorElement = document.createElement('div');
    selectorElement.innerHTML = selectorHTML;
    
    targetElement.appendChild(selectorElement.firstElementChild);
    
    // Add event listeners
    const buttons = selectorElement.querySelectorAll('.guide-selector-btn');
    buttons.forEach(button => {
      button.addEventListener('click', (e) => {
        const guideId = e.target.closest('.guide-selector-btn').dataset.guideId;
        this.showGuide(guideId);
      });
    });
  }

  // Show guide in modal or dedicated area
  showGuide(guideId) {
    const guide = this.getGuide(guideId);
    if (!guide) return;
    
    // Create modal or dedicated area
    const guideModal = document.createElement('div');
    guideModal.className = 'workflow-guide-modal';
    guideModal.innerHTML = `
      <div class="workflow-guide-modal-content">
        <div class="workflow-guide-modal-header">
          <h2>${guide.title}</h2>
          <button class="workflow-guide-close" aria-label="Close guide">
            <i class="bi bi-x-lg" aria-hidden="true"></i>
          </button>
        </div>
        
        <div class="workflow-guide-modal-body">
          ${this.createGuideHTML(guideId)}
        </div>
      </div>
    `;
    
    document.body.appendChild(guideModal);
    
    // Add event listeners
    const closeBtn = guideModal.querySelector('.workflow-guide-close');
    closeBtn.addEventListener('click', () => {
      guideModal.remove();
    });
    
    // Close on background click
    guideModal.addEventListener('click', (e) => {
      if (e.target === guideModal) {
        guideModal.remove();
      }
    });
    
    // Close on escape key
    document.addEventListener('keydown', function escapeHandler(e) {
      if (e.key === 'Escape') {
        guideModal.remove();
        document.removeEventListener('keydown', escapeHandler);
      }
    });
  }

  // Auto-detect page and show relevant guide
  autoShowGuide() {
    const currentPage = window.location.pathname;
    let guideId = null;
    
    // Map URLs to guide IDs
    const urlMap = {
      '/import-ecse': 'import-ecse',
      '/import-mileage': 'import-mileage',
      '/fleet': 'add-bus',
      '/students': 'add-student',
      '/assign-routes': 'assign-route',
      '/driver-dashboard': 'driver-log'
    };
    
    guideId = urlMap[currentPage];
    
    if (guideId) {
      // Add help button to page
      const helpButton = document.createElement('button');
      helpButton.className = 'workflow-help-button';
      helpButton.innerHTML = `
        <i class="bi bi-question-circle" aria-hidden="true"></i>
        <span>Need Help?</span>
      `;
      
      helpButton.addEventListener('click', () => {
        this.showGuide(guideId);
      });
      
      // Add to page
      document.body.appendChild(helpButton);
    }
  }

  // Static method to initialize
  static init() {
    const workflowGuides = new WorkflowGuides();
    
    // Auto-show relevant guides
    workflowGuides.autoShowGuide();
    
    // Store globally
    window.workflowGuides = workflowGuides;
    
    return workflowGuides;
  }
}

// Auto-initialize on DOM ready
document.addEventListener('DOMContentLoaded', () => {
  WorkflowGuides.init();
});

// Export for module use
if (typeof module !== 'undefined' && module.exports) {
  module.exports = WorkflowGuides;
}